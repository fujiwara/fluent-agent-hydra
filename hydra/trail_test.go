package hydra_test

import (
	"fmt"
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

	go fileWriter(file, Logs)

	ch := hydra.NewChannel()
	go hydra.Trail(file.Name(), "test", ch)

	resultCh := make(chan string)
	go reciever(t, ch, "test", resultCh)

	recieved := <-resultCh
	sent := strings.Join(Logs, "")
	if recieved != sent {
		t.Errorf("sent logs and recieved logs is different. sent %d bytes, recieved %d bytes", len(sent), len(recieved))
		fmt.Print(sent)
		fmt.Print(recieved)
	}
}

func fileWriter(file *os.File, logs []string) {
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
		if err != nil {
			log.Println("write failed", err)
		}
		randSleep()
	}
	file.Close()
}

func reciever(t *testing.T, ch chan *hydra.BulkMessage, tag string, resultCh chan string) {
	recieved := ""
	for {
		bulk := <-ch
		if bulk.Tag != "test" {
			t.Errorf("got %v\nwant %v", bulk.Tag, "test")
		}
		buf := string(*bulk.Buffer)
		log.Println("size:", len(buf), "buffer:", buf)
		recieved = recieved + buf + "\n"
		if strings.Index(buf, EOFMarker) != -1 {
			resultCh <- recieved
			return
		}
	}
}

func randSleep() {
	time.Sleep(time.Duration(rand.Int63n(int64(100))) * time.Millisecond)
}
