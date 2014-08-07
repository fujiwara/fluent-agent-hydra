package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"log"
	"time"
)

// OutForward ... recieve FluentRecordSet from channel, and send it to passed loggers until success.
func OutForward(messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat, loggers ...*fluent.Fluent) {
RECIEVE:
	for {
		recordSet, ok := <-messageCh
		if !ok {
			log.Println("[info] shutdown forward process")
			for _, logger := range loggers {
				logger.Shutdown()
			}
			return
		}
		first := true
		packed, err := recordSet.PackAsPacketForward()
		if err != nil {
			log.Println("[error] Can't create msgpack object", err)
			continue RECIEVE
		}
		for {
		LOGGER:
			for _, logger := range loggers {
				if logger.IsReconnecting() {
					continue LOGGER
				}
				err := logger.Send(packed)
				if err == nil {
					monitorCh <- &SentStat{
						Tag:      recordSet.Tag,
						Messages: int64(len(recordSet.Records)),
						Bytes:    int64(len(packed)),
					}
					continue RECIEVE // success
				}
				log.Println("[warning] Forwarding failed to", logger.Server, err)
			}
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
}

