package hydra_test

import (
	"bytes"
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"github.com/mattn/go-scan"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"testing"
)

func TestMonitorServer(t *testing.T) {
	config := &hydra.Config{
		MonitorAddress: ":0",
	}
	_, ch := hydra.NewChannel()
	monitor, _ := hydra.NewMonitor(config, ch)
	go monitor.Run()

	expectedMessages := make(map[string]int64)
	expectedBytes := make(map[string]int64)
	tags := []string{"foo", "bar", "dummy.test"}
	for _, tag := range tags {
		for i := 1; i <= 100; i++ {
			m := rand.Int63n(10)
			b := rand.Int63n(2560)
			ch <- &hydra.SentStat{
				Tag:      tag,
				Messages: m,
				Bytes:    b,
			}
			expectedMessages[tag] += m
			expectedBytes[tag] += b
		}
	}
	sleep(1)

	resp, err := http.Get(fmt.Sprintf("http://%s/", monitor.Addr))
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Error("invalid content-type", ct)
	}
	body, _ := ioutil.ReadAll(resp.Body)
	js := bytes.NewReader(body)
	for tag, n := range expectedMessages {
		js.Seek(int64(0), os.SEEK_SET)
		var got int64
		scan.ScanJSON(js, "/sent/"+tag+"/messages", &got)
		if got != n {
			t.Errorf("/sent/%s/messages got %d expected %d", tag, got, n)
		}
	}
	for tag, n := range expectedBytes {
		js.Seek(int64(0), os.SEEK_SET)
		var got int64
		scan.ScanJSON(js, "/sent/"+tag+"/bytes", &got)
		if got != n {
			t.Errorf("/sent/%s/bytes got %d expected %d", tag, got, n)
		}
	}

	log.Println(string(body))
}
