package hydra

import (
	"bytes"
	"encoding/json"
	"log"
	"runtime"
	"strings"
	"sync"
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

type Process interface {
	Run(*Context)
}

type Signal struct {
	message string
}

func (s Signal) Error() string {
	return s.message
}

func (s Signal) String() string {
	return s.message
}

func (s Signal) Signal() {
}

func NewSignal(message string) Signal {
	return Signal{message}
}

type Context struct {
	MessageCh     chan *fluent.FluentRecordSet
	MonitorCh     chan Stat
	ControlCh     chan interface{}
	InputProcess  sync.WaitGroup
	OutputProcess sync.WaitGroup
	StartProcess  sync.WaitGroup
}

func NewContext() *Context {
	return &Context{
		MessageCh: make(chan *fluent.FluentRecordSet, MessageChannelBufferLen),
		MonitorCh: make(chan Stat, MonitorChannelBufferLen),
		ControlCh: make(chan interface{}),
	}
}

func (c *Context) RunProcess(p Process) {
	c.StartProcess.Add(1)
	go p.Run(c)
}

func NewFluentRecordSet(tag, key string, format FileFormat, mod *RecordModifier, reg *Regexp, buffer []byte) *fluent.FluentRecordSet {
	t := time.Now()
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
			r.Timestamp = t
			if mod != nil {
				mod.Modify(r)
			}
			records = append(records, r)
		case FormatJSON:
			r := NewFluentRecordJSON(key, msg)
			r.Timestamp = t
			if mod != nil {
				mod.Modify(r)
			}
			records = append(records, r)
		case FormatRegexp:
			r := NewFluentRecordRegexp(key, msg, reg)
			r.Timestamp = t
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
		if col == "" {
			// ignore empty field
			continue
		}
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

func Run(config *Config) *Context {
	c := NewContext()

	if config.SubSecondTime {
		fluent.EnableEventTime = true
		log.Println("[info] SubSecondTime enabled. (for Fluentd 0.14 or later only!)")
	}

	if config.ReadBufferSize > 0 {
		ReadBufferSize = config.ReadBufferSize
		log.Println("[info] set ReadBufferSize", ReadBufferSize)
	}

	// start monitor server
	monitor, err := NewMonitor(config)
	if err != nil {
		log.Println("[error] Couldn't start monitor server.", err)
	} else {
		c.RunProcess(monitor)
	}

	// start out_forward
	outForward, err := NewOutForward(config.Servers)
	if err != nil {
		log.Println("[error]", err)
	} else {
		outForward.RoundRobin = config.ServerRoundRobin
		if outForward.RoundRobin {
			log.Println("[info] ServerRoundRobin enabled")
		}
		c.RunProcess(outForward)
	}

	// start watcher && in_tail
	if len(config.Logs) > 0 {
		watcher, err := NewWatcher()
		if err != nil {
			log.Println("[error]", err)
		}
		for _, configLogfile := range config.Logs {
			tail, err := NewInTail(configLogfile, watcher)
			if err != nil {
				log.Println("[error]", err)
			} else {
				c.RunProcess(tail)
			}
		}
		c.RunProcess(watcher)
	}

	// start in_forward
	if config.Receiver != nil {
		if runtime.GOMAXPROCS(0) < 2 {
			log.Println("[warning] When using Receiver, recommend to set GOMAXPROCS >= 2.")
		}
		inForward, err := NewInForward(config.Receiver)
		if err != nil {
			log.Println("[error]", err)
		} else {
			c.RunProcess(inForward)
		}
	}
	c.StartProcess.Wait()
	return c
}

func (c *Context) Shutdown() {
	close(c.ControlCh)
	c.InputProcess.Wait()
	close(c.MessageCh)
	c.OutputProcess.Wait()
}
