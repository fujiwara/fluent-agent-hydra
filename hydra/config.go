package hydra

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
)

const (
	DefaultFluentdPort = 24224
	DefaultFieldName   = "message"
)

type Config struct {
	TagPrefix      string
	FieldName      string
	Servers        []*ConfigServer
	Logs           []*ConfigLogfile
	MonitorAddress string
}

type ConfigServer struct {
	Host string
	Port int
}

type ConfigLogfile struct {
	Tag       string
	File      string
	FieldName string
}

func ReadConfig(filename string) (Config, error) {
	var config Config
	log.Println("[info] Loading config file:", filename)
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return config, err
	}
	config.Restrict()
	return config, nil
}

func (cs *ConfigServer) Restrict(c *Config) {
	if cs.Port == 0 {
		cs.Port = DefaultFluentdPort
	}
}

func (cs *ConfigServer) Address() string {
	return fmt.Sprintf("%s:%d", cs.Host, cs.Port)
}

func (cl *ConfigLogfile) Restrict(c *Config) {
	if cl.FieldName == "" {
		cl.FieldName = c.FieldName
	}
	if c.TagPrefix != "" {
		cl.Tag = c.TagPrefix + "." + cl.Tag
	}
}

func (c *Config) Restrict() {
	if c.FieldName == "" {
		c.FieldName = DefaultFieldName
	}
	for _, configServer := range c.Servers {
		configServer.Restrict(c)
	}
	for _, configLogfile := range c.Logs {
		configLogfile.Restrict(c)
	}
}
