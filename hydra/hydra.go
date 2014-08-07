package hydra

import (
	"bytes"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"time"
)

const (
	MessageChannelBufferLen = 1
	MonitorChannelBufferLen = 256
)

var (
	LineSeparator = []byte{'\n'}
)

type ShutdownType struct{
	message string
}

func (s *ShutdownType) Error() string { return s.message }

// NewChannel create channel for using by OutForward() and InTail().
func NewChannel() (chan *fluent.FluentRecordSet, chan Stat) {
	messageCh := make(chan *fluent.FluentRecordSet, MessageChannelBufferLen)
	monitorCh := make(chan Stat, MonitorChannelBufferLen)
	return messageCh, monitorCh
}

func NewFluentRecordSet(tag string, key string, buffer *[]byte) *fluent.FluentRecordSet {
	timestamp := uint64(time.Now().Unix())
	messages := bytes.Split(*buffer, LineSeparator)
	records := make([]fluent.TinyFluentRecord, len(messages))
	for i, message := range messages {
		records[i] = fluent.TinyFluentRecord{
			Timestamp: timestamp,
			Data:      map[string]interface{}{key: message},
		}
	}
	return &fluent.FluentRecordSet{
		Tag:     tag,
		Records: records,
	}
}
