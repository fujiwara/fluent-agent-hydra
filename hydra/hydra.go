package hydra

import (
	"bytes"
)

type BulkMessage struct {
	Tag      string
	Messages [][]byte
}

const (
	MessageChannelBufferLen = 1
	MonitorChannelBufferLen = 256
)

// NewChannel create channel for using by Forward() and Trail().
func NewChannel() (chan *BulkMessage, chan *Stat) {
	messageCh := make(chan *BulkMessage, MessageChannelBufferLen)
	monitorCh := make(chan *Stat, MonitorChannelBufferLen)
	return messageCh, monitorCh
}

func NewBulkMessage(tag string, buf *[]byte) *BulkMessage {
	return &BulkMessage{tag, bytes.Split(*buf, LineSeparator)}
}
