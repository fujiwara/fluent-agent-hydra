package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"log"
	"time"
)

// Forward ... recieve BulkMessages from channel, and send it to passed loggers until success.
func Forward(ch chan *BulkMessage, messageKey string, loggers ...*fluent.Fluent) {
RECIEVE:
	for {
		bulk, ok := <-ch
		if !ok {
			log.Println("shutdown forward process")
			for _, logger := range loggers {
				logger.Close()
			}
			return
		}
		tag := bulk.Tag
		messages := bulk.Messages
		first := true
		for {
		LOGGER:
			for _, logger := range loggers {
				if logger.IsReconnecting() {
					continue LOGGER
				}
				err := logger.PostBulkMessages(tag, messageKey, messages)
				if err == nil {
					continue RECIEVE // success
				}
				log.Println("Error on forwarding to", logger.FluentdAddr(), err)
			}
			if first {
				log.Printf(
					"All loggers are unavailable. pending %d messages tag:%s",
					len(messages),
					tag,
				)
			}
			time.Sleep(1 * time.Second) // waiting for any logger will be reconnected
		}
	}
}
