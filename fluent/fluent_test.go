package fluent_test

import (
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

func TestConnectToServer(t *testing.T) {
	hosts := []string{
		"127.0.0.1",
	}
	if os.Getenv("TRAVIS") != "" {
		// travis-ci has not ipv6 networking
		hosts = append(hosts, "[::1]")
	}
	for _, host := range hosts {
		port := startDummyServer(host)
		conf := fluent.Config{
			Server:  host + ":" + port,
			Timeout: time.Second * 1,
		}
		f, err := fluent.New(conf)
		if err != nil {
			t.Error(err)
		}
		if !f.Alive() {
			t.Error("server is not available")
		}
		f.Close()
		time.Sleep(time.Second * 1)
	}
}

func startDummyServer(host string) string {
	ch := make(chan string)
	go func() {
		l, err := net.Listen("tcp", host+":0")
		if err != nil {
			panic(err)
		}
		defer l.Close()
		addr := l.Addr().String()
		_, port, err := net.SplitHostPort(addr)
		if err != nil {
			panic(err)
		}
		ch <- port
		for {
			conn, _ := l.Accept()
			log.Println("acccept", conn.LocalAddr())
		}
	}()
	return <-ch
}
