package hydra

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"bytes"
	"log"
	"time"
)

// Forward recieve BulkMessages from channel, and send it to passed loggers until success.
func Forward(ch chan *BulkMessage, messageKey string, loggers ...*fluent.Fluent) {
RECIEVE_BLOCK:
	for {
		block := <-ch
		lines := bytes.Split(*block.Buffer, LineSeparator)
		for {
		LOGGER:
			for _, logger := range loggers {
				if logger.IsReconnecting() {
					continue LOGGER
				}
				err := logger.PostBulkMessages(block.Tag, messageKey, lines)
				if err == nil {
					continue RECIEVE_BLOCK // success
				}
				log.Println("forward", logger.FluentdAddr(), "error", err)
			}
			time.Sleep(1 * time.Second)
		}
	}
}
