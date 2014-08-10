package hydra_test

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"testing"
	"time"
)

var (
	EOFMarker      = "__EOF__"
	RotateMarker   = "__ROTATE__"
	TruncateMarker = "__TRUNCATE__"
	Logs           = []string{
		"single line\n",
		"multi line 1\nmulti line 2\nmultiline 3\n",
		"continuous line 1",
		"continuous line 2\n",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",   // 80 bytes == hydra.ReadBufferSize for testing
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\n",  // 81 bytes
		"ccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc\n", // 82byte
		"dddddddddddddddddddddddddddddddddddddddd",
		"ddddddddddddddddddddddddddddddddddddddd\n", // continuous line 80 bytes
		RotateMarker + "\n",
		"foo\n",
		"bar\n",
		"baz\n",
		TruncateMarker + "\n",
		"FOOOO\n",
		"BAAAR\n",
		"BAZZZZZZZ\n",
		EOFMarker + "\n",
	}
)

const (
	ReadBufferSizeForTest = 80
)

func TestTrail(t *testing.T) {
	hydra.ReadBufferSize = ReadBufferSizeForTest

	file, _ := ioutil.TempFile(os.TempDir(), "logfile.")
	file.WriteString("first line is must be trailed...\n")
	defer os.Remove(file.Name())

	go fileWriter(t, file, Logs)

	configLogfile := &hydra.ConfigLogfile{
		Tag:       "test",
		File:      file.Name(),
		FieldName: "message",
	}
	msgCh, monCh := hydra.NewChannel()
	watcher, err := hydra.NewWatcher()
	if err != nil {
		t.Error(err)
	}
	inTail, err := hydra.NewInTail(configLogfile, watcher, msgCh, monCh)
	if err != nil {
		t.Error(err)
	}
	go inTail.Run()
	go watcher.Run()

	resultCh := make(chan string)
	go reciever(t, msgCh, "test", resultCh)

	recieved := <-resultCh
	sent := strings.Join(Logs, "")
	if recieved != sent {
		t.Errorf("sent logs and recieved logs is different. sent %d bytes, recieved %d bytes", len(sent), len(recieved))
		fmt.Print(sent)
		fmt.Print(recieved)
	}
}

func fileWriter(t *testing.T, file *os.File, logs []string) {
	filename := file.Name()
	time.Sleep(1 * time.Second) // wait for start Tail...

	for _, line := range logs {
		if strings.Index(line, RotateMarker) != -1 {
			log.Println("fileWriter: rename file => file.old")
			os.Rename(filename, filename+".old")
			file.Close()
			file, _ = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
			log.Println("fileWriter: re-opened file")
		} else if strings.Index(line, TruncateMarker) != -1 {
			time.Sleep(1 * time.Second)
			log.Println("fileWriter: truncate(file, 0)")
			os.Truncate(filename, 0)
			file.Seek(int64(0), os.SEEK_SET)
		}
		_, err := file.WriteString(line)
		log.Print("fileWriter: wrote ", line)
		if err != nil {
			log.Println("write failed", err)
		}
		randSleep()
	}
	file.Close()
}

func reciever(t *testing.T, ch chan *fluent.FluentRecordSet, tag string, resultCh chan string) {
	recieved := ""
	for {
		recordSet := <-ch
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		for _, record := range recordSet.Records {
			message := record.Data["message"].([]byte)
			recieved = recieved + string(message) + string(hydra.LineSeparator)
			if strings.Index(string(message), EOFMarker) != -1 {
				resultCh <- recieved
				return
			}
		}
	}
}

func randSleep() {
	time.Sleep(time.Duration(rand.Int63n(int64(100))) * time.Millisecond)
}
