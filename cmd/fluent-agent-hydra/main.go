package main

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"log"
	"os"
)

const (
	defaultMessageKey = "message"
)

func main() {
	configFile := os.Args[1]
	done := make(chan bool)

	config, err := hydra.ReadConfig(configFile)
	if err != nil {
		log.Println("Can't load config", err)
		os.Exit(1)
	}

	loggers := make([]*fluent.Fluent, len(config.Servers))
	for i, server := range config.Servers {
		logger, err := fluent.New(fluent.Config{Server: server})
		if err != nil {
			log.Println("[warning] Can't initialize fluentd server.", err)
		}
		loggers[i] = logger
	}

	ch := hydra.NewChannel()
	for _, logfile := range config.Logs {
		var tag string
		if config.TagPrefix != "" {
			tag = config.TagPrefix + "." + logfile.Tag
		} else {
			tag = logfile.Tag
		}
		go hydra.Trail(logfile.File, tag, ch)
	}

	var fieldName string
	if config.FieldName != "" {
		fieldName = config.FieldName
	} else {
		fieldName = defaultMessageKey
	}

	go hydra.Forward(ch, fieldName, loggers...)

	<-done
}
