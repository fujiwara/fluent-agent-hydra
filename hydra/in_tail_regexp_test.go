package hydra_test

import (
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

var (
	RegexpLogs = []string{
		`192.168.0.1 - - [28/Feb/2013:12:00:00 +0900] "GET / HTTP/1.1" 200 777 "-" "Opera/12.0"` + "\n",
		"invalid regexp line\n",
	}
	RegexpParsed = []map[string]interface{}{
		{
			"user":    "-",
			"method":  "GET",
			"code":    int64(200),
			"size":    int64(777),
			"host":    "192.168.0.1",
			"path":    "/",
			"referer": "-",
			"agent":   "Opera/12.0",
			"time":    "28/Feb/2013:12:00:00 +0900",
			"_time":   time.Date(2013, time.February, 28, 12, 00, 00, 0, JST),
		},
		{"message": "invalid regexp line"},
	}
)

func TestTrailRegexp(t *testing.T) {
	tmpdir, _ := ioutil.TempDir(os.TempDir(), "hydra-test")
	file, _ := ioutil.TempFile(tmpdir, "logfile.")
	defer os.RemoveAll(tmpdir)

	reg := regexp.MustCompile(`^(?P<host>[^ ]*) [^ ]* (?P<user>[^ ]*) \[(?P<time>[^\]]*)\] "(?P<method>\S+)(?: +(?P<path>[^\"]*?)(?: +\S*)?)?" (?P<code>[^ ]*) (?P<size>[^ ]*)(?: "(?P<referer>[^\"]*)" "(?P<agent>[^\"]*)")?$`)
	configLogfile := &hydra.ConfigLogfile{
		Tag:        "test",
		File:       file.Name(),
		Format:     hydra.FormatRegexp,
		Regexp:     &hydra.Regexp{Regexp: reg},
		FieldName:  "message",
		ConvertMap: hydra.NewConvertMap("size:integer,code:integer"),
		TimeParse:  true,
		TimeFormat: "02/Jan/2006:15:04:05 -0700",
		TimeKey:    "time",
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
		fileWriter(t, file, RegexpLogs)
	}()

	i := 0
	for i < len(RegexpLogs) {
		recordSet := <-msgCh
		if recordSet.Tag != "test" {
			t.Errorf("got %v\nwant %v", recordSet.Tag, "test")
		}
		for _, _record := range recordSet.Records {
			record := _record.(*fluent.TinyFluentRecord)
			d := record.GetAllData()
			e := RegexpParsed[i]
			if ts, ok := e["_time"]; ok {
				if ts.(time.Time).Unix() != record.Timestamp {
					t.Errorf("expected record[%d] timestamp %s got %s", i, ts, record.Timestamp)
				}
				delete(e, "_time")
			}
			if !reflect.DeepEqual(e, d) {
				t.Errorf("expected %#v got %#v", d, e)
			}
			i++
		}
	}
}
