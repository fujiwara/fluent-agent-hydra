package fluent_test

import (
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

var (
	tag   = "test.tag"
	ts    = time.Unix(1417269412, 0)
	key   = "testkey"
	value = []byte("datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue ")
)

func BenchmarkTinyFluentMessage(b *testing.B) {
	for i := 0; i < b.N; i++ {
		msg := &fluent.TinyFluentMessage{ts, key, value}
		packed, _ := msg.Pack()
		b.SetBytes(int64(len(packed)))
	}
}

func BenchmarkTinyFluentRecord(b *testing.B) {
	for i := 0; i < b.N; i++ {
		msg := &fluent.TinyFluentRecord{ts, map[string]interface{}{key: value}}
		packed, _ := msg.Pack()
		b.SetBytes(int64(len(packed)))
	}
}
