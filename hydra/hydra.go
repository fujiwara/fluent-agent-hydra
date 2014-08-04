package hydra

import (
	"bytes"
)

type BulkMessage struct {
	Tag      string
	Messages [][]byte
}

const (
	ChannelBufferLen = 1
)

// NewChannel create channel for using by Forward() and Trail().
func NewChannel() chan *BulkMessage {
	return make(chan *BulkMessage, ChannelBufferLen)
}

func NewBulkMessage(tag string, buf *[]byte) *BulkMessage {
	return &BulkMessage{tag, bytes.Split(*buf, LineSeparator)}
}
