package hydra_test

import (
	"strings"
	"testing"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func createRecordsetSample(n int) string {
	data := `{"foo":"1","bar":"2","hoge":"3","fuga":"4","hoga":"5","hobar":"6","time":"2015-10-29T10:17:45+09:00"}
`
	return strings.TrimRight(strings.Repeat(data, n), "\n")
}

func TestNewFluentRecordSetLTSV(t *testing.T) {
	buf := []byte(createRecordsetSample(1))
	record := hydra.NewFluentRecordSetLTSV("dummy", "message", nil, buf)
	if record.Tag != "dummy" {
		t.Errorf("invalid tag: %s", record.Tag)
	}
	if len(record.Records) != 1 {
		t.Errorf("invalid record length: %#v", len(record.Records))
	}
}

func TestNewFluentRecordSetJSON(t *testing.T) {
	buf := []byte(createRecordsetSample(1))
	record := hydra.NewFluentRecordSetJSON("dummy", "message", nil, buf)
	if record.Tag != "dummy" {
		t.Errorf("invalid tag: %s", record.Tag)
	}
	if len(record.Records) != 1 {
		t.Errorf("invalid record length: %#v", len(record.Records))
	}
}

func BenchmarkNewFluentRecordSetLTSV(b *testing.B) {
	b.ResetTimer()
	buf := []byte(createRecordsetSample(10))
	for i := 0; i < b.N; i++ {
		_ = hydra.NewFluentRecordSetLTSV("dummy", "message", nil, buf)
	}
}

func BenchmarkNewFluentRecordSetJSON(b *testing.B) {
	b.ResetTimer()
	buf := []byte(createRecordsetSample(10))
	for i := 0; i < b.N; i++ {
		_ = hydra.NewFluentRecordSetJSON("dummy", "message", nil, buf)
	}
}
