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
	LTSVLogs = []string{
		"foo:1\tbar:2\n",
		"foo:123\n",
		"foo\tbar:baz\n",
	}
	LTSVParsed = []map[string]interface{}{
		{"foo": 1, "bar": "2"},
		{"foo": 123},
		{"bar": "baz"},
	}
)

func TestTrailLTSV(t *testing.T) {
	file, _ := ioutil.TempFile(os.TempDir(), "logfile.")
	defer os.Remove(file.Name())

	configLogfile := &hydra.ConfigLogfile{
		Tag:        "test",
		File:       file.Name(),
		Format:     hydra.LTSV,
		ConvertMap: hydra.NewConvertMap("foo:integer"),
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
		fileWriter(t, file, LTSVLogs)
	}()

	i := 0
	for i < len(LTSVLogs) {
		recordSet := <-msgCh
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		i += len(recordSet.Records)
		for j, _record := range recordSet.Records {
			record := _record.(*fluent.TinyFluentRecord)
			if foo, _ := record.GetData("foo"); foo != LTSVParsed[j]["foo"] {
				t.Errorf("unexpected record %v", record)
			}
			if bar, _ := record.GetData("bar"); bar != LTSVParsed[j]["bar"] {
				t.Errorf("unexpected record %v", record)
			}
		}
	}
}
