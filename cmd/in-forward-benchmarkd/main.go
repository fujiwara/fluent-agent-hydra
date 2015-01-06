package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

var (
	DEBUG     = false
	MessageCh chan fluent.FluentRecordType
)

func main() {
	var port int
	flag.IntVar(&port, "port", 24224, "listen port")
	flag.IntVar(&port, "p", 24224, "listen port")
	flag.BoolVar(&DEBUG, "d", false, "debug(print accepted record)")
	flag.BoolVar(&DEBUG, "debug", false, "debug(print accepted record)")
	flag.Parse()
	counter := int64(0)
	_, ch := runServer(port, &counter)
	go reporter(&counter)
	<-ch
}

func reporter(counter *int64) {
	c := time.Tick(1 * time.Second)
	prev := int64(0)
	for _ = range c {
		n := atomic.LoadInt64(counter)
		if prev == n {
			continue
		}
		log.Printf("[info] %d messages\n", n-prev)
		prev = n
	}
}

func runServer(port int, counter *int64) (string, chan bool) {
	addr := fmt.Sprintf(":%d", port)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[info] server listing", l.Addr())
	ch := make(chan bool)
	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				log.Println("[error] accept error", err)
			}
			go handleConn(conn, counter)
		}
	}()
	return l.Addr().String(), ch
}

func handleConn(conn net.Conn, counter *int64) {
	for {
		recordSets, err := fluent.DecodeEntries(conn)
		if err == io.EOF {
			conn.Close()
			return
		} else if err != nil {
			log.Println("decode entries failed", err, conn.LocalAddr())
			conn.Close()
			return
		}
		n := 0
		for _, recordSet := range recordSets {
			n += len(recordSet.Records)
			if !DEBUG {
				continue
			}
			for _, record := range recordSet.Records {
				if DEBUG {
					fmt.Println(record)
				}
			}
		}
		atomic.AddInt64(counter, int64(n))
	}
}
