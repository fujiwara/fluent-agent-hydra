package hydra_test

import (
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"testing"
	"fmt"
)

func TestReadConfig(t *testing.T) {
	config, err := hydra.ReadConfig("./config_example.toml")
	if err != nil {
		t.Error("read config failed", err)
	}
	fmt.Printf("%#v\n", config)
	if config.TagPrefix != "foo" {
		t.Error("invalid TagPrefix got", config.TagPrefix, "expected", "foo")
	}
	if config.FieldName != "msg" {
		t.Error("invalid FieldName got", config.FieldName, "expected", "msg")
	}
	if len(config.Servers) != 2 || config.Servers[0] != "127.0.0.1:24224" || config.Servers[1] != "127.0.0.1:24225" {
		t.Errorf("invalid Servers got %#v", config.Servers)
	}
	if len(config.Logs) != 2 {
		t.Errorf("invalid Logs got %#v", config.Logs)
	}
	if config.Logs[0].Tag != "tag1" || config.Logs[0].File != "/tmp/foo.log" {
		t.Errorf("invalid Logs[0] got %#v", config.Logs[0])
	}
	if config.Logs[1].Tag != "tag2" || config.Logs[1].File != "/tmp/bar.log" {
		t.Errorf("invalid Logs[1] got %#v", config.Logs[1])
	}
}
