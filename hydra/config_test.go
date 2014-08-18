package hydra_test

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"testing"
)

func TestReadConfig(t *testing.T) {
	config, err := hydra.ReadConfig("./config_test.toml")
	if err != nil {
		t.Error("read config failed", err)
	}
	fmt.Printf("%#v\n", config)
	if config.TagPrefix != "foo" {
		t.Error("invalid TagPrefix got", config.TagPrefix, "expected", "foo")
	}
	if config.FieldName != "message" {
		t.Error("invalid FieldName got", config.FieldName, "expected", "msg")
	}
	if len(config.Servers) != 2 {
		t.Errorf("invalid Servers got %#v", config.Servers)
	}
	if config.Servers[0].Host != "127.0.0.1" || config.Servers[0].Port != 24224 {
		t.Errorf("invalid Servers[0] got %#v", config.Servers[0])
	}
	if config.Servers[1].Host != "127.0.0.1" || config.Servers[1].Port != 24225 {
		t.Errorf("invalid Servers[1] got %#v", config.Servers[1])
	}
	if len(config.Logs) != 2 {
		t.Errorf("invalid Logs got %#v", config.Logs)
	}
	if config.Logs[0].Tag != "foo.tag1" || config.Logs[0].File != "/tmp/foo.log" || config.Logs[0].FieldName != "message" {
		t.Errorf("invalid Logs[0] got %#v", config.Logs[0])
	}
	if config.Logs[1].Tag != "foo.tag2" || config.Logs[1].File != "/tmp/bar.log" || config.Logs[1].FieldName != "msg" {
		t.Errorf("invalid Logs[1] got %#v", config.Logs[1])
	}

	if config.Receiver.Host != "localhost" || config.Receiver.Port != 24224 {
		t.Errorf("invalid Receiver got %#v", config.Receiver)
	}

	if config.Monitor.Host != "127.0.0.2" || config.Monitor.Port != 24223 {
		t.Errorf("invalid Monitor got %#v", config.Monitor)
	}
}
