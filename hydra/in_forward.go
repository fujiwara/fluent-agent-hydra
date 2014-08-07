package hydra

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"io"
	"log"
	"net"
)

func InForward(config *ConfigReceiver, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) (net.Addr, error) {
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Println("[error]", err)
		return nil, err
	}
	log.Println("[info] Server listing", l.Addr())
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("[error] accept error", err)
			}
			go inForwardHandleConn(conn, messageCh, monitorCh)
		}
	}()
	return l.Addr(), nil
}

func inForwardHandleConn(conn net.Conn, messageCh chan *fluent.FluentRecordSet, monitorCh chan Stat) {
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
			messageCh <- &recordSet
		}
	}
}
