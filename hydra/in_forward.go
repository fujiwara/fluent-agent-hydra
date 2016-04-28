package hydra

import (
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

const (
	FlashInterval = 200 * time.Millisecond
)

type InForward struct {
	index        int
	listener     net.Listener
	Addr         net.Addr
	messageCh    chan *fluent.FluentRecordSet
	monitorCh    chan Stat
	messageQueue *MessageQueue
}

func NewInForward(config *ConfigReceiver) (*InForward, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[error]", err)
		return nil, err
	}
	log.Println("[info] Receiver listing", l.Addr())
	f := &InForward{
		listener:     l,
		Addr:         l.Addr(),
		messageQueue: NewMessageQueue(config.MaxBufferMessages),
	}
	return f, nil
}

func (f *InForward) Run(c *Context) {
	c.InputProcess.Add(1)
	defer c.InputProcess.Done()
	f.messageCh = c.MessageCh
	f.monitorCh = c.MonitorCh

	c.StartProcess.Done()

	f.monitorCh <- &ReceiverStat{
		Address:           f.Addr.String(),
		MaxBufferMessages: int64(f.messageQueue.maxMessages),
	}

	go f.feed()
	go func() {
		<-c.ControlCh
		f.listener.Close()
	}()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			if strings.Index(err.Error(), "use of closed network connection") != -1 {
				log.Println("[info] shutdown in_forward accept")
				// closed
				return
			} else {
				log.Println("[error] accept error", err)
			}
			continue
		}
		f.monitorCh <- &ReceiverStat{
			Connections: 1,
		}
		go f.handleConn(conn, c)
	}
}

func (f *InForward) feed() {
	for {
		if rs, ok := f.messageQueue.Dequeue(); ok {
			f.messageCh <- rs
		} else {
			<-time.After(FlashInterval)
			f.monitorCh <- &ReceiverStat{
				Buffered: int64(f.messageQueue.Len()),
			}
		}
	}
}

func (f *InForward) handleConn(conn net.Conn, c *Context) {
	c.InputProcess.Add(1)
	defer c.InputProcess.Done()
	defer func() {
		f.monitorCh <- &ReceiverStat{
			Connections: -1,
		}
	}()

	for {
		select {
		case <-c.ControlCh:
			log.Println("shutdown in_forward connection", conn)
			return
		default:
		}
		recordSets, err := fluent.DecodeEntries(conn)
		if err == io.EOF {
			conn.Close()
			return
		} else if err != nil {
			log.Println("[error] Decode entries failed", err, conn.LocalAddr())
			conn.Close()
			return
		}
		m := int64(0)
		d := int64(0)
		for _, recordSet := range recordSets {
			rs := &recordSet
			d += f.messageQueue.Enqueue(rs)
			m += int64(len(rs.Records))
		}
		f.monitorCh <- &ReceiverStat{
			Messages: m,
			Disposed: d,
			Buffered: int64(f.messageQueue.Len()),
		}
	}
}
