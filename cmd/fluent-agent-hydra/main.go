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
	tag := os.Args[1]
	filenames := os.Args[2:]
	log.Println("tag", tag, "filenames", filenames)

	done := make(chan bool)
	loggerPrimary, err := fluent.New(fluent.Config{
		FluentPort: 24224,
		FluentHost: "127.0.0.1",
	})
	loggerSecondary, err := fluent.New(fluent.Config{
		FluentPort: 24225,
		FluentHost: "127.0.0.1",
	})
	if err != nil {
		log.Println("logger initialize failed", err)
	}
	ch := hydra.NewChannel()
	for _, filename := range filenames {
		go hydra.Trail(filename, tag, ch)
	}
	go hydra.Forward(ch, defaultMessageKey, loggerPrimary, loggerSecondary)

	<-done
}
