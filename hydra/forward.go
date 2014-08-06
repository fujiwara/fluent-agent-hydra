package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"log"
	"time"
)

// Forward ... recieve BulkMessages from channel, and send it to passed loggers until success.
func Forward(messageCh chan *BulkMessage, monitorCh chan *Stat, messageKey string, loggers ...*fluent.Fluent) {
RECIEVE:
	for {
		bulk, ok := <-messageCh
		if !ok {
			log.Println("[info] shutdown forward process")
			for _, logger := range loggers {
				logger.Shutdown()
			}
			return
		}
		tag := bulk.Tag
		messages := bulk.Messages
		first := true
		packed, err := fluent.NewPackedForwardObject(tag, messageKey, messages)
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
					monitorCh <- &Stat{
						Tag:      tag,
						Messages: int64(len(messages)),
						Bytes:    int64(len(packed)),
					}
					continue RECIEVE // success
				}
				log.Println("[warning] Forwarding failed to", logger.Server, err)
			}
			if first {
				log.Printf(
					"[warning] All servers are unavailable. pending %d messages tag:%s",
					len(messages),
					tag,
				)
				first = false
			}
			time.Sleep(1 * time.Second) // waiting for any logger will be reconnected
		}
	}
}
