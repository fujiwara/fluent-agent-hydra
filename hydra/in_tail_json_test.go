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
		`{"foo":"1","bar":"2"}` + "\n",
		`{"foo":"123"}` + "\n",
		`{"bar":"baz"}` + "\n",
		"invalid JSON line\n",
	}
	JSONParsed = []map[string]interface{}{
		{"foo": "1", "bar": "2"},
		{"foo": "123"},
		{"bar": "baz"},
		{"message": "invalid JSON line"},
	}
)

func TestTrailJSON(t *testing.T) {
	file, _ := ioutil.TempFile(os.TempDir(), "logfile.")
	defer os.Remove(file.Name())

	configLogfile := &hydra.ConfigLogfile{
		Tag:       "test",
		File:      file.Name(),
		Format:    hydra.JSON,
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
				t.Errorf("unexpected record got:%#v expected:%#v", record, JSONParsed[i])
			}
			if bar, _ := record.GetData("bar"); bar != JSONParsed[i]["bar"] {
				t.Errorf("unexpected record got:%#v expected:%#v", record, JSONParsed[i])
			}
			i++
		}
	}
}
