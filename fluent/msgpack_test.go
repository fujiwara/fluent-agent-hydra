package fluent_test

import (
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/ugorji/go/codec"
	"testing"
)

var (
	tag   = "test.tag"
	ts    = int64(1417269412)
	key   = "testkey"
	value = []byte("datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue datavalue ")
	mh    codec.MsgpackHandle
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
