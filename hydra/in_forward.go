package hydra

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"io"
	"log"
	"net"
	"time"
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

func NewInForward(config *ConfigReceiver, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (*InForward, error) {
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
		messageCh:    messageCh,
		monitorCh:    monitorCh,
		messageQueue: NewMessageQueue(config.MaxBufferMessages),
	}
	monitorCh <- &ReceiverStat{
		Address:           f.Addr.String(),
		MaxBufferMessages: int64(config.MaxBufferMessages),
	}
	return f, nil
}

func (f *InForward) Run() {
	go f.feed()
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			log.Println("[error] accept error", err)
		}
		f.monitorCh <- &ReceiverStat{
			Connections: 1,
		}
		go f.handleConn(conn)
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

func (f *InForward) handleConn(conn net.Conn) {
	defer func() {
		f.monitorCh <- &ReceiverStat{
			Connections: -1,
		}
	}()
	for {
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
