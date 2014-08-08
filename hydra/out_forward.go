package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"log"
	"time"
)

type OutForward struct {
	loggers   []*fluent.Fluent
	messageCh chan *fluent.FluentRecordSet
	monitorCh chan Stat
}

const (
	serverHealthCheckInterval = 3 * time.Second
)

// OutForward ... recieve FluentRecordSet from channel, and send it to passed loggers until success.
func NewOutForward(configServers []*ConfigServer, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (*OutForward, error) {
	loggers := make([]*fluent.Fluent, len(configServers))
	for i, server := range configServers {
		logger, err := fluent.New(fluent.Config{Server: server.Address()})
		if err != nil {
			log.Println("[warning]", err)
		} else {
			log.Println("[info] server", server.Address())
		}
		loggers[i] = logger
		logger.Send([]byte{})
	}
	return &OutForward{
		loggers:   loggers,
		messageCh: messageCh,
		monitorCh: monitorCh,
	}, nil
}

func (f *OutForward) Run() {
	for i, _ := range f.loggers {
		go f.checkServerHealth(i)
	}
	for {
		err := f.outForwardRecieve()
		if err != nil {
			if _, ok := err.(*ShutdownType); ok {
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
		return &ShutdownType{"Shutdown forward process"}
	}
	first := true
	packed, err := recordSet.PackAsPackedForward()
	if err != nil {
		return err
	}
	for {
	LOGGER:
		for _, logger := range f.loggers {
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
