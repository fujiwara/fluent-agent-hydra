package hydra_test

import (
	"log"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
	client "github.com/t-k/fluent-logger-golang/fluent"
)

func TestInForward(t *testing.T) {
	config := &hydra.ConfigReceiver{
		Host:              "127.0.0.1",
		Port:              0,
		MaxBufferMessages: 1000,
	}
	c := hydra.NewContext()
	inForward, err := hydra.NewInForward(config)
	if err != nil {
		t.Error(err)
	}
	c.RunProcess(inForward)

	host, _port, _ := net.SplitHostPort(inForward.Addr.String())
	port, _ := strconv.Atoi(_port)
	logger, err := client.New(client.Config{
		FluentHost: host,
		FluentPort: port,
	})
	if err != nil {
		t.Error(err)
	}
	log.Println("logger", logger)
	defer logger.Close()

	tag := "myapp.access"
	for i := 0; i < 10; i++ {
		var data = map[string]interface{}{
			"foo":  "bar",
			"hoge": "hoge",
		}
		logger.Post(tag, data)
	}
	n := 0
RECEIVE:
	for {
		select {
		case <-c.MessageCh:
			n++
			continue
		case <-time.After(1 * time.Second):
			break RECEIVE
		}
	}
	if n != 10 {
		t.Errorf("arrived messages %d expected %d", n, 10)
	}
}
