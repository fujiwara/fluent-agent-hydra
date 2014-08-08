package hydra_test

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

var (
	TestTag          = "test"
	TestFieldName    = "message"
	TestMessageLines = []string{"message1", "message2", "message3"}
)

func sleep(n int) {
	time.Sleep(time.Duration(n) * time.Second)
}

func prepareRecordSet() *fluent.FluentRecordSet {
	message := strings.Join(TestMessageLines, "\n")
	messageBytes := []byte(message)
	return hydra.NewFluentRecordSet(TestTag, TestFieldName, &messageBytes)
}

func newConfigServer(addr string) *hydra.ConfigServer {
	host, _port, _ := net.SplitHostPort(addr)
	port, _ := strconv.Atoi(_port)
	return &hydra.ConfigServer{
		Host: host,
		Port: port,
	}
}

func TestForwardSingle(t *testing.T) {
	log.Println("---- TestForwardSingle ----")
	counter := int64(0)

	addr, mockCloser := runMockServer(t, "", &counter)
	configServer := newConfigServer(addr)
	msgCh, monCh := hydra.NewChannel()
	outForward, err := hydra.NewOutForward([]*hydra.ConfigServer{configServer}, msgCh, monCh)
	if err != nil {
		t.Error(err)
	}
	go outForward.Run()

	recordSet := prepareRecordSet()
	msgCh <- recordSet
	sleep(3)

	if n := atomic.LoadInt64(&counter); n != int64(len(TestMessageLines)) {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines), "recieved", n)
	}
	close(msgCh)
	close(mockCloser)
	sleep(1)
}

func TestForwardReconnect(t *testing.T) {
	log.Println("---- TestForwardReconnect ----")
	counter := int64(0)

	addr, mockCloser := runMockServer(t, "", &counter)
	configServer := newConfigServer(addr)
	msgCh, monCh := hydra.NewChannel()
	outForward, err := hydra.NewOutForward([]*hydra.ConfigServer{configServer}, msgCh, monCh)
	if err != nil {
		t.Error(err)
	}
	go outForward.Run()

	recordSet := prepareRecordSet()
	msgCh <- recordSet
	sleep(1)

	t.Log("notify shutdown mockServer")
	close(mockCloser)
	t.Log("waiting for shutdown complated 3 sec")
	sleep(3)

	t.Log("restarting mock server on same addr", addr)
	_, mockCloser = runMockServer(t, addr, &counter)
	sleep(1)
	msgCh <- recordSet // Afeter unexpected server closing, first Write() will be succeeded and lost...
	sleep(1)
	msgCh <- recordSet
	t.Log("waiting for reconnect & resend completed 5 sec")
	sleep(3)

	if n := atomic.LoadInt64(&counter); n != int64(len(TestMessageLines)*2) {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines)*2, "recieved", n)
	}
	close(mockCloser)
	close(msgCh)
	sleep(1)
}

func TestForwardFailOver(t *testing.T) {
	log.Println("---- TestForwardFailOver ----")
	counter := int64(0)

	primaryAddr, primaryCloser := runMockServer(t, "", &counter)
	close(primaryCloser) // shutdown primary server immediately
	sleep(1)
	primaryConfigServer := newConfigServer(primaryAddr)
	secondaryAddr, secondaryCloser := runMockServer(t, "", &counter)
	secondaryConfigServer := newConfigServer(secondaryAddr)
	configServers := []*hydra.ConfigServer{
		primaryConfigServer,
		secondaryConfigServer,
	}
	msgCh, monCh := hydra.NewChannel()
	outForward, err := hydra.NewOutForward(configServers, msgCh, monCh)
	if err != nil {
		t.Error(err)
		return
	}
	go outForward.Run()

	sleep(1)

	recordSet := prepareRecordSet()
	msgCh <- recordSet
	sleep(1)

	if n := atomic.LoadInt64(&counter); n != int64(len(TestMessageLines)) {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines), "recieved", n)
	}
	close(msgCh)
	close(secondaryCloser)
	sleep(1)
}

func runMockServer(t *testing.T, addr string, counter *int64) (string, chan bool) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	log.Println("[info][mockServer] listing", l.Addr())
	ch := make(chan bool)
	connections := make([]net.Conn, 0)
	go func() {
		for {
			select {
			case <-ch:
				log.Println("[info][mockServer] shutdown mock server", l.Addr())
				for _, conn := range connections {
					err := conn.Close()
					log.Println("[info][mockServer] closing connection to", conn.RemoteAddr(), err)
				}
				l.Close()
				return
			default:
			}
			l.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second))
			conn, err := l.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				log.Println("[error][mockServer] accept error", err)
			}
			connections = append(connections, conn)
			log.Println("[info][mockServer] accepting new connection from", conn.RemoteAddr())
			go handleConn(conn, t, counter)
		}
	}()
	return l.Addr().String(), ch
}

func handleConn(conn net.Conn, t *testing.T, counter *int64) {
	for {
		recordSets, err := fluent.DecodeEntries(conn)
		if err != nil {
			if err != io.EOF {
				log.Println("[error][mockServer] decode entries failed", err, conn.LocalAddr())
			}
			conn.Close()
			return
		}
		for _, recordSet := range recordSets {
			for _, record := range recordSet.Records {
				atomic.AddInt64(counter, int64(1))
				log.Println(record)
			}
		}
	}
}
