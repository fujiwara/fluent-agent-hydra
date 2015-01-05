package hydra

import (
	"bytes"
	"strings"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

const (
	MessageChannelBufferLen = 1
	MonitorChannelBufferLen = 256
	LineSeparatorStr        = "\n"
	LTSVColSeparatorStr     = "\t"
	LTSVDataSeparatorStr    = ":"
)

var (
	LineSeparator = []byte{'\n'}
)

type ShutdownType struct {
	message string
}

func (s *ShutdownType) Error() string { return s.message }

// NewChannel create channel for using by OutForward() and InTail().
func NewChannel() (chan *fluent.FluentRecordSet, chan Stat) {
	messageCh := make(chan *fluent.FluentRecordSet, MessageChannelBufferLen)
	monitorCh := make(chan Stat, MonitorChannelBufferLen)
	return messageCh, monitorCh
}

func NewFluentRecordSet(tag string, key string, buffer []byte) *fluent.FluentRecordSet {
	timestamp := time.Now().Unix()
	messages := bytes.Split(buffer, LineSeparator)
	records := make([]fluent.FluentRecordType, len(messages))
	for i, m := range messages {
		records[i] = &fluent.TinyFluentMessage{
			Timestamp: timestamp,
			FieldName: key,
			Message:   m,
		}
	}
	return &fluent.FluentRecordSet{
		Tag:     tag,
		Records: records,
	}
}

func NewFluentRecordSetLTSV(tag string, buffer []byte) *fluent.FluentRecordSet {
	timestamp := time.Now().Unix()
	lines := strings.Split(string(buffer), LineSeparatorStr)
	records := make([]fluent.FluentRecordType, len(lines))
	for i, line := range lines {
		data := make(map[string]interface{})
		for _, col := range strings.Split(line, LTSVColSeparatorStr) {
			pair := strings.SplitN(col, LTSVDataSeparatorStr, 2)
			if len(pair) < 2 {
				continue
			}
			data[pair[0]] = pair[1]
		}
		records[i] = &fluent.TinyFluentRecord{
			Timestamp: timestamp,
			Data:      data,
		}
	}
	return &fluent.FluentRecordSet{
		Tag:     tag,
		Records: records,
	}
}
