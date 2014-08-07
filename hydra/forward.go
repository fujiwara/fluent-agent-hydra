package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"log"
	"time"
)

// OutForward ... recieve FluentRecordSet from channel, and send it to passed loggers until success.
func OutForward(configServers []*ConfigServer, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) {

	loggers := make([]*fluent.Fluent, len(configServers))
	for i, server := range configServers {
		logger, err := fluent.New(fluent.Config{Server: server.Address()})
		if err != nil {
			log.Println("[warning] Can't initialize fluentd server.", err)
		} else {
			log.Println("[info] server", server.Address())
		}
		loggers[i] = logger
		logger.Send([]byte{})
		go checkServerHealth(i, logger, monitorCh)
	}
	for {
		err := outForwardRecieve(messageCh, monitorCh, loggers...)
		if err != nil {
			if _, ok := err.(*ShutdownType); ok {
				log.Println("[info]", err)
				return
			} else {
				log.Println("[error]", err)
			}
		}
	}
	panic("xxx")
}

func outForwardRecieve(messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat, loggers ...*fluent.Fluent) error {
	recordSet, ok := <-messageCh
	if !ok {
		for _, logger := range loggers {
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
		for _, logger := range loggers {
			if logger.IsReconnecting() {
				continue LOGGER
			}
			err := logger.Send(packed)
			if err != nil {
				log.Println("[error]", err)
				continue LOGGER
			}
			monitorCh <- &SentStat{
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

func checkServerHealth(i int, logger *fluent.Fluent, monitorCh chan Stat) {
	c := time.Tick(3 * time.Second)
	for _ = range c {
		monitorCh <- &ServerStat{
			Index:   i,
			Address: logger.Server,
			Alive:   logger.Alive(),
			Error:   logger.LastErrorString(),
		}
	}
}
