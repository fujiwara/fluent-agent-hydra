package hydra_test

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

var (
	JST, _   = time.LoadLocation("Asia/Tokyo")
	LTSVLogs = []string{
		strings.Join([]string{"foo:1", "bar:2", "time:2015-05-26T11:22:33Z"}, "\t") + "\n",
		strings.Join([]string{"foo:AAA", "bar:BBB", "time:2013-11-22T04:21:31+09:00"}, "\t") + "\n",
		"foo:123\n",
		strings.Join([]string{"foo", "bar:baz"}, "\t") + "\n",
		"invalid LTSV line\n",
	}
	LTSVParsed = []map[string]interface{}{
		{"foo": int64(1), "bar": "2", "_time": time.Date(2015, time.May, 26, 11, 22, 33, 0, time.UTC)},
		{"foo": "AAA", "bar": "BBB", "_time": time.Date(2013, time.November, 22, 04, 21, 31, 0, JST)},
		{"foo": int64(123)},
		{"bar": "baz"},
		{"message": "invalid LTSV line"},
	}
)

func TestTrailLTSV(t *testing.T) {
	tmpdir, _ := ioutil.TempDir(os.TempDir(), "hydra-test")
	file, _ := ioutil.TempFile(tmpdir, "logfile.")
	defer os.RemoveAll(tmpdir)

	configLogfile := &hydra.ConfigLogfile{
		Tag:        "test",
		File:       file.Name(),
		Format:     hydra.FormatLTSV,
		ConvertMap: hydra.NewConvertMap("foo:integer"),
		FieldName:  "message",
		TimeParse:  true,
		TimeFormat: hydra.DefaultTimeFormat,
		TimeKey:    hydra.DefaultTimeKey,
	}
	c := hydra.NewContext()
	watcher, err := hydra.NewWatcher()
	if err != nil {
		t.Error(err)
	}
	inTail, err := hydra.NewInTail(configLogfile, watcher)
	if err != nil {
		t.Error(err)
	}
	c.RunProcess(inTail)
	c.RunProcess(watcher)
	go func() {
		time.Sleep(1 * time.Second)
		fileWriter(t, file, LTSVLogs)
	}()

	i := 0
	for i < len(LTSVLogs) {
		recordSet := <-c.MessageCh
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		for _, _record := range recordSet.Records {
			record := _record.(*fluent.TinyFluentRecord)
			if foo, _ := record.GetData("foo"); foo != LTSVParsed[i]["foo"] {
				t.Errorf("unexpected record %v", record)
			}
			if bar, _ := record.GetData("bar"); bar != LTSVParsed[i]["bar"] {
				t.Errorf("unexpected record %v", record)
			}
			if ts, ok := LTSVParsed[i]["_time"]; ok {
				if !ts.(time.Time).Equal(record.Timestamp) {
					t.Errorf("expected timestamp %s got %s", ts, record.Timestamp)
				}
			}
			i++
		}
	}
}
