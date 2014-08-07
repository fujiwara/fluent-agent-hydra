package hydra

import (
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"net"
	"strconv"
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
	Receivers      []*ConfigReceiver
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

type ConfigReceiver struct {
	Host string
	Port int
}

func ReadConfig(filename string) (*Config, error) {
	var config Config
	log.Println("[info] Loading config file:", filename)
	if _, err := toml.DecodeFile(filename, &config); err != nil {
		return nil, err
	}
	config.Restrict()
	return &config, nil
}

func NewConfigByArgs(args []string, fieldName string, monitorAddr string) *Config {
	tag := args[0]
	file := args[1]
	servers := args[2:]

	configLogfile := &ConfigLogfile{
		Tag:       tag,
		File:      file,
		FieldName: fieldName,
	}
	configLogfiles := []*ConfigLogfile{configLogfile}

	configServers := make([]*ConfigServer, len(servers))
	for i, server := range servers {
		var port int
		host, _port, err := net.SplitHostPort(server)
		if err != nil {
			host = server
			port = DefaultFluentdPort
		} else {
			port, _ = strconv.Atoi(_port)
		}
		configServers[i] = &ConfigServer{
			Host: host,
			Port: port,
		}
	}
	config := &Config{
		FieldName:      fieldName,
		Servers:        configServers,
		Logs:           configLogfiles,
		MonitorAddress: monitorAddr,
	}
	config.Restrict()
	return config
}

func (cs *ConfigServer) Restrict(c *Config) {
	if cs.Port == 0 {
		cs.Port = DefaultFluentdPort
	}
}

func (cr *ConfigReceiver) Restrict(c *Config) {
	if cr.Port == 0 {
		cr.Port = DefaultFluentdPort
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
	for _, subconf := range c.Servers {
		subconf.Restrict(c)
	}
	for _, subconf := range c.Logs {
		subconf.Restrict(c)
	}
	for _, subconf := range c.Receivers {
		subconf.Restrict(c)
	}
}
