package hydra_test

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

var (
	JSONLogs = []string{
		`{"foo":"1","bar":"2","time":"2014-12-31T12:00:01+09:00"}` + "\n",
		`{"foo":"123","time":"2015-04-29T00:00:00Z"}` + "\n",
		`{"bar":"baz"}` + "\n",
		"invalid JSON line\n",
	}
	JSONParsed = []map[string]interface{}{
		{"foo": "1", "bar": "2", "_time": time.Date(2014, time.December, 31, 12, 00, 01, 0, JST)},
		{"foo": "123", "_time": time.Date(2015, time.April, 29, 00, 00, 00, 0, time.UTC)},
		{"bar": "baz"},
		{"message": "invalid JSON line"},
	}
)

func TestTrailJSON(t *testing.T) {
	tmpdir, _ := ioutil.TempDir(os.TempDir(), "hydra-test")
	file, _ := ioutil.TempFile(tmpdir, "logfile.")
	defer os.RemoveAll(tmpdir)

	configLogfile := &hydra.ConfigLogfile{
		Tag:        "test",
		File:       file.Name(),
		Format:     hydra.JSON,
		FieldName:  "message",
		TimeParse:  true,
		TimeFormat: hydra.DefaultTimeFormat,
		TimeKey:    hydra.DefaultTimeKey,
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
	go func() {
		time.Sleep(1 * time.Second)
		fileWriter(t, file, JSONLogs)
	}()

	i := 0
	for i < len(JSONLogs) {
		recordSet := <-msgCh
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		for _, _record := range recordSet.Records {
			record := _record.(*fluent.TinyFluentRecord)
			if foo, _ := record.GetData("foo"); foo != JSONParsed[i]["foo"] {
				t.Errorf("unexpected record got:foo=%#v expected:%#v", foo, JSONParsed[i]["foo"])
			}
			if bar, _ := record.GetData("bar"); bar != JSONParsed[i]["bar"] {
				t.Errorf("unexpected record got:bar=%#v expected:%#v", bar, JSONParsed[i]["bar"])
			}
			if ts, ok := JSONParsed[i]["_time"]; ok {
				if ts.(time.Time).Unix() != record.Timestamp {
					t.Errorf("expected timestamp %s got %s", ts, record.Timestamp)
				}
			}
			i++
		}
	}
}
