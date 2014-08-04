package hydra_test

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"github.com/ugorji/go/codec"
	"io"
	"log"
	"math"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	TestTag          = "test"
	TestMessageKey   = "message"
	TestMessageLines = []string{"message1", "message2", "message3"}
	numOfMessages    = 0
)

func _TestForwardSingle(t *testing.T) {
	numOfMessages = 0
	addr, mockCloser := runMockServer(t, "")
	logger, err := fluent.New(fluent.Config{Server: addr})
	if err != nil {
		t.Errorf("can't create logger to %s", addr, err)
	}

	ch := hydra.NewChannel()
	go hydra.Forward(ch, TestMessageKey, logger)

	message := strings.Join(TestMessageLines, "\n")
	messageBytes := []byte(message)
	bulk := hydra.NewBulkMessage(TestTag, &messageBytes)
	ch <- bulk
	time.Sleep(3 * time.Second)

	if numOfMessages != len(TestMessageLines) {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines), "recieved", numOfMessages)
	}
	close(ch)
	close(mockCloser)
}

func TestForwardReconnect(t *testing.T) {
	numOfMessages = 0
	addr, mockCloser := runMockServer(t, "")
	logger, err := fluent.New(fluent.Config{Server: addr})
	if err != nil {
		t.Error("can't create logger to %s", addr, err)
	}
	ch := hydra.NewChannel()
	go hydra.Forward(ch, TestMessageKey, logger)

	message := strings.Join(TestMessageLines, "\n")
	messageBytes := []byte(message)
	bulk := hydra.NewBulkMessage(TestTag, &messageBytes)
	ch <- bulk
	time.Sleep(1 * time.Second)

	t.Log("notify shutdown mockServer")
	close(mockCloser)
	t.Log("waiting for shutdown complated 3 sec")
	time.Sleep(3 * time.Second)

	t.Log("restarting mock server on same addr", addr)
	_, _ = runMockServer(t, addr)
	ch <- bulk // Afeter unexpected server closing, first Write() will be succeeded and lost...
	ch <- bulk
	t.Log("waiting for reconnect & resend completed 5 sec")
	time.Sleep(5 * time.Second)

	if numOfMessages != len(TestMessageLines)*2 {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines)*2, "recieved", numOfMessages)
	}
}

func TestForwardFailOver(t *testing.T) {
	numOfMessages = 0
	loggers := make([]*fluent.Fluent, 2)
	mockClosers := make([]chan bool, 2)

	for i := 0; i < 2; i++ {
		addr, mockCloser := runMockServer(t, "")
		if i == 0 {
			close(mockCloser) // shutdown primary server immediately
		}
		logger, err := fluent.New(fluent.Config{Server: addr})
		if i == 0 && err == nil {
			t.Error("create logger must return err (server is down)")
		} else if i == 1 && err != nil {
			t.Error("create logger failed", err)
		}
		loggers[i] = logger
		mockClosers[i] = mockCloser
	}
	time.Sleep(1 * time.Second)

	ch := hydra.NewChannel()
	go hydra.Forward(ch, TestMessageKey, loggers...)
	time.Sleep(1 * time.Second)

	message := strings.Join(TestMessageLines, "\n")
	messageBytes := []byte(message)
	bulk := hydra.NewBulkMessage(TestTag, &messageBytes)
	ch <- bulk
	time.Sleep(1 * time.Second)

	if numOfMessages != len(TestMessageLines) {
		t.Error("insufficient recieved messages. sent", len(TestMessageLines), "recieved", numOfMessages)
	}
	close(ch)
	close(mockClosers[1])
}

func runMockServer(t *testing.T, addr string) (string, chan bool) {
	if addr == "" {
		addr = "127.0.0.1:0"
	}
	l, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("mock server listing", l.Addr())
	ch := make(chan bool)
	connections := make([]net.Conn, 0)
	go func() {
		for {
			select {
			case <-ch:
				log.Println("[info] shutdown mock server", l.Addr())
				for _, conn := range connections {
					err := conn.Close()
					log.Println("[info] closing connection to", conn.RemoteAddr(), err)
				}
				l.Close()
				return
			default:
			}
			l.(*net.TCPListener).SetDeadline(time.Now().Add(1e9))
			conn, err := l.Accept()
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue
				}
				log.Println("[error] accept error", err)
			}
			connections = append(connections, conn)
			log.Println("[info] accepting new connection from", conn.RemoteAddr())
			go handleConn(conn, t)
		}
	}()

	return l.Addr().String(), ch
}

func handleConn(conn net.Conn, t *testing.T) {
	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	dec := codec.NewDecoder(conn, &mh)

	for i, line := range TestMessageLines {
		v := []interface{}{nil, nil, nil}
		err := dec.Decode(&v)
		if err != nil {
			if err == io.EOF {
				return
			}
			t.Error("decode error", err)
			return
		}

		// tag
		tag, ok := v[0].([]byte)
		if !ok || string(tag) != TestTag {
			t.Error("invalid tag", v[0])
		}

		// timestamp
		ts, ok := v[1].(uint64)
		if !ok {
			t.Error("invalid timestamp", v[1])
		}
		now := time.Now().Unix()
		if math.Abs(float64(now)-float64(ts)) > 5 {
			t.Error("invalid timestamp", ts, "now", now)
		}

		// message body
		message, ok := v[2].(map[string]interface{})
		if !ok || message[TestMessageKey] == nil {
			t.Error("invalid message", v[2])
		}
		messageBytes, ok := message[TestMessageKey].([]byte)
		if !ok || string(messageBytes) != string(line) {
			t.Error("invalid message string got", message, "expected", line)
		}
		numOfMessages++
		//t.Log("recieved", conn.LocalAddr(), i, string(tag), ts, string(messageBytes))
		fmt.Println("recieved", conn.LocalAddr(), i, string(tag), ts, string(messageBytes))
	}
}
