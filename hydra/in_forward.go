package hydra

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"io"
	"log"
	"net"
)

type InForward struct {
	listener  net.Listener
	Addr      net.Addr
	messageCh chan *fluent.FluentRecordSet
	monitorCh chan Stat
}

func NewInForward(config *ConfigReceiver, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (*InForward, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[error]", err)
		return nil, err
	}
	log.Println("[info] Server listing", l.Addr())
	return &InForward{
		listener:  l,
		Addr:      l.Addr(),
		messageCh: messageCh,
		monitorCh: monitorCh,
	}, nil
}

func (f *InForward) Run() {
	for {
		conn, err := f.listener.Accept()
		if err != nil {
			log.Println("[error] accept error", err)
		}
		go f.inForwardHandleConn(conn)
	}
}

func (f *InForward) inForwardHandleConn(conn net.Conn) {
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
		for _, recordSet := range recordSets {
			f.messageCh <- &recordSet
		}
	}
}
