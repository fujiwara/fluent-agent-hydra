package hydra

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"github.com/BurntSushi/toml"
)

const (
	DefaultFluentdPort       = 24224
	DefaultFieldName         = "message"
	DefaultMaxBufferMessages = 1024 * 1024
	DefaultTimeKey           = "time"
	DefaultTimeFormat        = time.RFC3339
)

type Config struct {
	TagPrefix        string
	FieldName        string
	ReadBufferSize   int
	Servers          []*ConfigServer
	ServerRoundRobin bool
	Logs             []*ConfigLogfile
	Receiver         *ConfigReceiver
	Monitor          *ConfigMonitor
}

type ConfigServer struct {
	Host string
	Port int
}

type ConfigLogfile struct {
	Tag        string
	File       string
	FieldName  string
	Format     FileFormat
	ConvertMap ConvertMap `toml:"Types"`
	TimeParse  bool
	TimeKey    string
	TimeFormat string
}

type ConfigReceiver struct {
	Host              string
	Port              int
	MaxBufferMessages int
}

type ConfigMonitor struct {
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
		FieldName: fieldName,
		Servers:   configServers,
		Logs:      configLogfiles,
	}

	if monitorAddr != "" {
		host, _port, err := net.SplitHostPort(monitorAddr)
		if err != nil {
			log.Println("[error] invalid monitor address", monitorAddr, "disabled.")
		} else {
			port, _ := strconv.Atoi(_port)
			config.Monitor = &ConfigMonitor{
				Host: host,
				Port: port,
			}
		}
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
	switch cr.MaxBufferMessages {
	case 0:
		cr.MaxBufferMessages = DefaultMaxBufferMessages
	case -1:
		cr.MaxBufferMessages = 0
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
	if cl.TimeKey == "" {
		cl.TimeKey = DefaultTimeKey
	}
	if cl.TimeFormat == "" {
		cl.TimeFormat = DefaultTimeFormat
	}
}

func (cl *ConfigLogfile) IsStdin() bool {
	return cl.File == StdinFilename
}

func (cr *ConfigMonitor) Restrict(c *Config) {
	if cr.Host == "" {
		cr.Host = DefaultMonitorHost
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
	if c.Receiver != nil {
		c.Receiver.Restrict(c)
	}
}
