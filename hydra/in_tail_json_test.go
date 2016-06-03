package hydra_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

var (
	JSONLogs = []string{
		`{"foo":"1","bar":"2","time":"2014-12-31T12:00:01+09:00"}` + "\n",
		`{"foo":"1","bar":2,"time":"2014-12-31T12:00:01+09:00"}` + "\n",
		`{"foo":"123","time":"2015-04-29T00:00:00Z"}` + "\n",
		`{"bar":"baz"}` + "\n",
		"invalid JSON line\n",
	}
	JSONParsed = []map[string]interface{}{
		{
			"foo":   "1",
			"bar":   int64(2),
			"time":  "2014-12-31T12:00:01+09:00",
			"_time": time.Date(2014, time.December, 31, 12, 00, 01, 0, JST),
		},
		{
			"foo":   "1",
			"bar":   int64(2),
			"time":  "2014-12-31T12:00:01+09:00",
			"_time": time.Date(2014, time.December, 31, 12, 00, 01, 0, JST),
		},
		{
			"foo":   "123",
			"time":  "2015-04-29T00:00:00Z",
			"_time": time.Date(2015, time.April, 29, 00, 00, 00, 0, time.UTC),
		},
		{
			"bar": "baz",
		},
		{
			"message": "invalid JSON line",
		},
	}
)

func TestTrailJSON(t *testing.T) {
	tmpdir, _ := ioutil.TempDir(os.TempDir(), "hydra-test")
	file, _ := ioutil.TempFile(tmpdir, "logfile.")
	defer os.RemoveAll(tmpdir)

	configLogfile := &hydra.ConfigLogfile{
		Tag:        "test",
		File:       file.Name(),
		Format:     hydra.FormatJSON,
		FieldName:  "message",
		ConvertMap: hydra.NewConvertMap("bar:integer"),
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
		fileWriter(t, file, JSONLogs)
	}()

	i := 0
	for i < len(JSONLogs) {
		recordSet := <-c.MessageCh
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		for _, _record := range recordSet.Records {
			record := _record.(*fluent.TinyFluentRecord)
			d := record.GetAllData()
			e := JSONParsed[i]
			if ts, ok := e["_time"]; ok {
				if !ts.(time.Time).Equal(record.Timestamp) {
					t.Errorf("expected record[%d] timestamp %s got %s", i, ts, record.Timestamp)
				}
				delete(e, "_time")
			}
			if !reflect.DeepEqual(e, d) {
				t.Errorf("expected %#v got %#v", e, d)
			}
			i++
		}
	}
}
