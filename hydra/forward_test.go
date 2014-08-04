package hydra_test

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"github.com/ugorji/go/codec"
	"log"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	TestTag          = "test"
	TestMessageKey   = "message"
	TestMessageLines = []string{"message1", "message2", "message3"}
)

func TestForwardSingle(t *testing.T) {
	addr, mockCloser := runMockServer(t)
	addrs := strings.Split(addr.String(), ":")
	host := addrs[0]
	port, _ := strconv.Atoi(addrs[1])

	logger, err := fluent.New(fluent.Config{
		FluentHost: host,
		FluentPort: port,
	})
	if err != nil {
		t.Error("can't create logger to", addr, err)
	}

	ch := hydra.NewChannel()
	go hydra.Forward(ch, TestMessageKey, logger)

	message := strings.Join(TestMessageLines, "\n")
	messageBytes := []byte(message)
	bulk := hydra.NewBulkMessage(TestTag, &messageBytes)
	ch <- bulk
	close(ch)
	close(mockCloser)
	time.Sleep(3 * time.Second)
}

func runMockServer(t *testing.T) (net.Addr, chan bool) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	log.Println("mock server listing", l.Addr())
	ch := make(chan bool)
	go func() {
		for {
			select {
			case <-ch:
				log.Println("shutdown mock server")
				return
			default:
				conn, err := l.Accept()
				if err != nil {
					// handle error
					continue
				}
				go handleConn(conn, t)
			}
		}
	}()
	return l.Addr(), ch
}

func handleConn(conn net.Conn, t *testing.T) {
	var mh codec.MsgpackHandle
	mh.MapType = reflect.TypeOf(map[string]interface{}(nil))
	dec := codec.NewDecoder(conn, &mh)

	for i, line := range TestMessageLines {
		v := []interface{}{nil, nil, nil}
		err := dec.Decode(&v)
		if err != nil {
			t.Error("err", err)
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
		log.Println(i, string(tag), ts, string(messageBytes))
	}
}
