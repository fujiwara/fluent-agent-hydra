package main

import (
	"flag"
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

const (
	DEBUG = false
)

func main() {
	var port int
	flag.IntVar(&port, "port", 24224, "listen port")
	flag.IntVar(&port, "p", 24224, "listen port")
	flag.Parse()
	counter := int64(0)
	_, ch := runServer(port, &counter)
	go func() {
		c := time.Tick(1 * time.Second)
		prev := int64(0)
		for _ = range c {
			n := atomic.LoadInt64(&counter)
			if prev == n {
				continue
			}
			log.Printf("[info] %d messages\n", n-prev)
			prev = n
		}
	}()
	<-ch
}

func runServer(port int, counter *int64) (string, chan bool) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
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
			for _, record := range recordSet.Records {
				n++
				if DEBUG {
					log.Println(record)
				}
			}
		}
		atomic.AddInt64(counter, int64(n))
	}
}
