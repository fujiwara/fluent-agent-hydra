package main

import (
	"fmt"
	"os"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage\ntail-f filename")
		os.Exit(1)
	}
	filename, err := hydra.Rel2Abs(os.Args[1])
	if err != nil {
		panic(err)
	}
	watcher, err := hydra.NewWatcher()
	if err != nil {
		panic(err)
	}
	messageCh, monitorCh := hydra.NewChannel()
	config := &hydra.ConfigLogfile{
		Tag:       "dummy",
		File:      filename,
		FieldName: "message",
	}
	inTail, err := hydra.NewInTail(config, watcher, messageCh, monitorCh)
	go watcher.Run()
	go inTail.Run()
	for {
		recordSet := <-messageCh
		for _, record := range recordSet.Records {
			b, ok := record.GetData("message")
			if ok {
				fmt.Println(string(b.([]byte)))
			}
		}
	}
}
