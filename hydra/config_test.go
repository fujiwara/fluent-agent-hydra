package hydra_test

import (
	"fmt"
	"testing"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
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
	if config.ReadBufferSize != 1024 {
		t.Error("invalid ReadBufferSize got", config.ReadBufferSize)
	}
	if config.ServerRoundRobin != true {
		t.Error("invalid ServerRoundRobin got", config.ServerRoundRobin)
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

	if len(config.Logs) != 5 {
		t.Errorf("invalid Logs got %#v", config.Logs)
	}
	if c := config.Logs[0]; c.Tag != "foo.tag1" ||
		c.File != "/tmp/foo.log" ||
		c.FieldName != "message" ||
		c.TimeParse != false {
		t.Errorf("invalid Logs[0] got %#v", c)
	}

	if c := config.Logs[1]; c.Tag != "foo.tag2" ||
		c.File != "/tmp/bar.log" ||
		c.FieldName != "msg" ||
		c.TimeParse != false {
		t.Errorf("invalid Logs[1] got %#v", c)
	}

	if c := config.Logs[2]; c.Tag != "foo.ltsv" ||
		c.File != "/tmp/baz.log" ||
		c.TimeParse != true ||
		c.TimeKey != "time" ||
		c.TimeFormat != "2006-01-02T15:04:05Z07:00" {
		t.Errorf("invalid Logs[2] got %#v", c)
	}

	if c := config.Logs[3]; c.Tag != "foo.ltsv" ||
		c.File != "/tmp/bazz.log" ||
		c.TimeParse != true ||
		c.TimeKey != "timestamp" ||
		c.TimeFormat != "02/Jan/2006:15:04:05 Z0700" {
		t.Errorf("invalid Logs[3] got %#v", c)
	}

	if c := config.Logs[4]; c.Tag != "foo.regexp" ||
		c.File != "/tmp/regexp.log" ||
		c.Format != hydra.FormatRegexp ||
		c.TimeParse != true ||
		c.TimeKey != "time" ||
		c.TimeFormat != hydra.TimeFormatApache {
		t.Errorf("invalid Logs[4] got %#v", c)
	}

	if config.Receiver.Host != "localhost" || config.Receiver.Port != 24224 {
		t.Errorf("invalid Receiver got %#v", config.Receiver)
	}

	if config.Monitor.Host != "127.0.0.2" || config.Monitor.Port != 24223 {
		t.Errorf("invalid Monitor got %#v", config.Monitor)
	}
}
