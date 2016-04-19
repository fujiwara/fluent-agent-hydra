package hydra

import (
	"bytes"
	"encoding/json"
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
	StdinFilename           = "-"
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

func NewFluentRecordSet(tag, key string, format FileFormat, mod *RecordModifier, reg *Regexp, buffer []byte) *fluent.FluentRecordSet {
	t := time.Now().Unix()
	messages := bytes.Split(buffer, LineSeparator)
	records := make([]fluent.FluentRecordType, 0, len(messages))
	for _, msg := range messages {
		switch format {
		default:
			r := &fluent.TinyFluentMessage{
				Timestamp: t,
				FieldName: key,
				Message:   msg,
			}
			records = append(records, r)
		case FormatLTSV:
			r := NewFluentRecordLTSV(key, msg)
			if mod != nil {
				mod.Modify(r)
			}
			records = append(records, r)
		case FormatJSON:
			r := NewFluentRecordJSON(key, msg)
			if mod != nil {
				mod.Modify(r)
			}
			records = append(records, r)
		case FormatRegexp:
			r := NewFluentRecordRegexp(key, msg, reg)
			if mod != nil {
				mod.Modify(r)
			}
			records = append(records, r)
		}
	}
	return &fluent.FluentRecordSet{
		Tag:     tag,
		Records: records,
	}
}

func NewFluentRecordLTSV(key string, line []byte) *fluent.TinyFluentRecord {
	s := string(line)
	data := make(map[string]interface{})
	for _, col := range strings.Split(s, LTSVColSeparatorStr) {
		pair := strings.SplitN(col, LTSVDataSeparatorStr, 2)
		if len(pair) == 2 {
			data[pair[0]] = pair[1]
		} else {
			// invalid LTSV format.
			data[key] = s
		}
	}
	return &fluent.TinyFluentRecord{Data: data}
}

func NewFluentRecordJSON(key string, line []byte) *fluent.TinyFluentRecord {
	data := make(map[string]interface{})
	err := json.Unmarshal(line, &data)
	if err != nil {
		data[key] = string(line)
	}
	return &fluent.TinyFluentRecord{Data: data}
}

func NewFluentRecordRegexp(key string, line []byte, r *Regexp) *fluent.TinyFluentRecord {
	s := string(line)
	data := make(map[string]interface{})
	if match := r.FindStringSubmatch(s); match == nil {
		data[key] = s
	} else {
		for i, name := range r.SubexpNames() {
			if i != 0 {
				data[name] = match[i]
			}
		}
	}
	return &fluent.TinyFluentRecord{Data: data}
}
