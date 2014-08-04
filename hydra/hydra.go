package hydra

import (
	"bytes"
)

type BulkMessage struct {
	Tag    string
	Buffer *[]byte
}

const (
	ChannelBufferLen = 1
)

// NewChannel create channel for using by Forward() and Trail().
func NewChannel() chan *BulkMessage {
	return make(chan *BulkMessage, ChannelBufferLen)
}

func (b *BulkMessage) Messages() [][]byte {
	return bytes.Split(*b.Buffer, LineSeparator)
}
