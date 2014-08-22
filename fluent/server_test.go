package fluent_test

import (
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
	if _, err := tinyRecord.Pack(); err != nil {
		t.Error(err)
	}
	tinyMessage := &fluent.TinyFluentMessage{
		Timestamp: now,
		FieldName: "message",
		Message:   []byte("text"),
	}
	if _, err := tinyMessage.Pack(); err != nil {
		t.Error(err)
	}
}
