package fluent_test

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"testing"
	"time"
)

func TestPack(t *testing.T) {
	now := time.Now().Unix()
	tinyRecord := &fluent.TinyFluentRecord{
		Timestamp: now,
		Data:      map[string]interface{}{"message": "text"},
	}
	packedTinyRecord, err := tinyRecord.Pack()
	if err != nil {
		t.Error(err)
	}
	tinyMessage := &fluent.TinyFluentMessage{
		Timestamp: now,
		FieldName: "message",
		Message:   []byte("text"),
	}
	packedTinyMessage, err := tinyMessage.Pack()
	if err != nil {
		t.Error(err)
	}
}
