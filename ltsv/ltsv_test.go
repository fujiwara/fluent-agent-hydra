package ltsv_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/fujiwara/fluent-agent-hydra/ltsv"
)

var testMap = map[string]interface{}{
	"foo":   1,
	"bar":   "BAR",
	"baz":   "B\tZ",
	"bytes": []byte{65, 66, 67},
}
var compareLTSV = "bar:BAR\tbaz:B\\tZ\tbytes:ABC\tfoo:1\n"

func TestLTSVEncode(t *testing.T) {
	buf := &bytes.Buffer{}
	encoder := ltsv.NewEncoder(buf)
	encoder.Encode(testMap)
	if string(buf.Bytes()) != compareLTSV {
		t.Error("unexpected encoded", string(buf.Bytes()))
	}
}

func BenchmarkLTSVEncode(b *testing.B) {
	buf := &bytes.Buffer{}
	encoder := ltsv.NewEncoder(buf)
	for i := 0; i < b.N; i++ {
		encoder.Encode(testMap)
	}
}

func BenchmarkJSONEncode(b *testing.B) {
	buf := &bytes.Buffer{}
	encoder := json.NewEncoder(buf)
	for i := 0; i < b.N; i++ {
		encoder.Encode(testMap)
	}
}
