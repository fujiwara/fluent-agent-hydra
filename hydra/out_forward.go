package hydra

import (
	"log"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

type OutForward struct {
	loggers    []*fluent.Fluent
	messageCh  chan *fluent.FluentRecordSet
	monitorCh  chan Stat
	sent       int64
	RoundRobin bool
}

const (
	serverHealthCheckInterval = 3 * time.Second
	maxKeepAliveSentCount     = 100
)

// OutForward ... recieve FluentRecordSet from channel, and send it to passed loggers until success.
func NewOutForward(configServers []*ConfigServer) (*OutForward, error) {
	loggers := make([]*fluent.Fluent, len(configServers))
	for i, server := range configServers {
		logger, err := fluent.New(fluent.Config{Server: server.Address()})
		if err != nil {
			log.Println("[warning]", err)
		} else {
			log.Println("[info] Server", server.Address(), "connected")
		}
		loggers[i] = logger
		logger.Send([]byte{})
	}
	return &OutForward{
		loggers: loggers,
		sent:    0,
	}, nil
}

func (f *OutForward) Run(c *Context) {
	c.OutputProcess.Add(1)
	defer c.OutputProcess.Done()
	f.messageCh = c.MessageCh
	f.monitorCh = c.MonitorCh

	c.StartProcess.Done()

	for i, _ := range f.loggers {
		go f.checkServerHealth(i)
	}

	for {
		err := f.outForwardRecieve()
		if err != nil {
			if _, ok := err.(Signal); ok {
				log.Println("[info]", err)
				return
			} else {
				log.Println("[error]", err)
			}
		}
	}
}

func (f *OutForward) outForwardRecieve() error {
	recordSet, ok := <-f.messageCh
	if !ok {
		for _, logger := range f.loggers {
			logger.Shutdown()
		}
		return Signal{"shutdown out_forward"}
	}
	first := true
	packed, err := recordSet.PackAsPackedForward()
	if err != nil {
		return err
	}
	nLoggers := int64(len(f.loggers))
	for {
		var loggers []*fluent.Fluent
		if f.RoundRobin {
			index := f.sent % nLoggers
			loggers = f.loggers[index:nLoggers]
			loggers = append(loggers, f.loggers[0:index]...)
		} else {
			loggers = f.loggers
		}
	LOGGER:
		for _, logger := range loggers {
			if logger.IsReconnecting() {
				continue LOGGER
			}
			err := logger.Send(packed)
			if err != nil {
				log.Println("[error]", err)
				continue LOGGER
			}
			f.monitorCh <- &SentStat{
				Tag:      recordSet.Tag,
				Messages: int64(len(recordSet.Records)),
				Bytes:    int64(len(packed)),
				Sents:    1,
			}
			f.sent++
			if logger.Sent%maxKeepAliveSentCount == 0 {
				logger.RefreshConnection()
			}
			return nil // success
		}
		// all loggers seems down...
		if first {
			log.Printf(
				"[warning] All servers are unavailable. pending %d messages tag:%s",
				len(recordSet.Records),
				recordSet.Tag,
			)
			first = false
		}
		time.Sleep(1 * time.Second) // waiting for any logger will be reconnected
	}
}

func (f *OutForward) checkServerHealth(i int) {
	c := time.Tick(serverHealthCheckInterval)
	for _ = range c {
		f.monitorCh <- &ServerStat{
			Index:   i,
			Address: f.loggers[i].Server,
			Alive:   f.loggers[i].Alive(),
			Error:   f.loggers[i].LastErrorString(),
		}
	}
}
