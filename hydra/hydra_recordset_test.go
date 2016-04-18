package hydra_test

import (
	"strings"
	"testing"

	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func createRecordsetSampleJSON(n int) string {
	data := `{"foo":"1","bar":"2","hoge":"3","fuga":"4","hoga":"5","hobar":"6","time":"2015-10-29T10:17:45+09:00"}
`
	return strings.TrimRight(strings.Repeat(data, n), "\n")
}

func createRecordsetSampleLTSV(n int) string {
	data := `foo:1	bar:2	hoge:3	fuga:4	hoga:5	hobar:6	time:2015-10-29T10:17:45+09:00
`
	return strings.TrimRight(strings.Repeat(data, n), "\n")
}

func TestNewFluentRecordSetLTSV(t *testing.T) {
	buf := []byte(createRecordsetSampleLTSV(1))
	record := hydra.NewFluentRecordSet("dummy", "message", hydra.FormatLTSV, nil, nil, buf)
	if record.Tag != "dummy" {
		t.Errorf("invalid tag: %s", record.Tag)
	}
	if len(record.Records) != 1 {
		t.Errorf("invalid record length: %#v", len(record.Records))
	}
}

func TestNewFluentRecordSetJSON(t *testing.T) {
	buf := []byte(createRecordsetSampleJSON(1))
	record := hydra.NewFluentRecordSet("dummy", "message", hydra.FormatJSON, nil, nil, buf)
	if record.Tag != "dummy" {
		t.Errorf("invalid tag: %s", record.Tag)
	}
	if len(record.Records) != 1 {
		t.Errorf("invalid record length: %#v", len(record.Records))
	}
}

func BenchmarkNewFluentRecordSetLTSV(b *testing.B) {
	b.ResetTimer()
	buf := []byte(createRecordsetSampleLTSV(10))
	for i := 0; i < b.N; i++ {
		_ = hydra.NewFluentRecordSet("dummy", "message", hydra.FormatLTSV, nil, nil, buf)
	}
}

func BenchmarkNewFluentRecordSetJSON(b *testing.B) {
	b.ResetTimer()
	buf := []byte(createRecordsetSampleJSON(10))
	for i := 0; i < b.N; i++ {
		_ = hydra.NewFluentRecordSet("dummy", "message", hydra.FormatJSON, nil, nil, buf)
	}
}
