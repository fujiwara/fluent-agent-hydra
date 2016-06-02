package hydra_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/fujiwara/fluent-agent-hydra/fluent"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
)

func newDummyRecordSet(n int) *fluent.FluentRecordSet {
	records := make([]fluent.FluentRecordType, n)
	ts := time.Now()
	for i := 0; i < n; i++ {
		data := make(map[string]interface{})
		data["message"] = []byte(fmt.Sprintf("message%d", i))
		records[i] = &fluent.TinyFluentRecord{
			Timestamp: ts,
			Data:      data,
		}
	}
	rs := &fluent.FluentRecordSet{
		Tag:     "dummy",
		Records: records,
	}
	return rs

}

func TestMessageQueue(t *testing.T) {
	n := 10
	queue := hydra.NewMessageQueue(55)
	for i := 1; i <= n; i++ {
		rs := newDummyRecordSet(i)
		queue.Enqueue(rs)
	}
	if queue.Len() != 55 {
		t.Errorf("invalid queue.Len() %d expected 55", queue.Len())
	}

	// [1 2 3 4 5 6 7 8 9 10] + 1 => disposed=[1]
	d := queue.Enqueue(newDummyRecordSet(1))
	if queue.Len() != 55 {
		t.Errorf("invalud queue.Len() %d", queue.Len())
	}
	if d != 1 {
		t.Errorf("invalid disposed %d", d)
	}

	// [2 3 4 5 6 7 8 9 10 1] + 10 => disposed=[2 3 4 5]
	d = queue.Enqueue(newDummyRecordSet(10))
	if queue.Len() != 51 {
		t.Errorf("invalud queue.Len() %d", queue.Len())
	}
	if d != 14 {
		t.Errorf("invalid disposed %d", d)
	}

	// [6 7 8 9 10 1 10] => [7 8 9 10 1 10]
	rs, ok := queue.Dequeue()
	if !ok || len(rs.Records) != 6 {
		t.Error("invalid dequeued rs", rs)
	}
	if queue.Len() != 45 {
		t.Error("invalid dequeued rs", rs)
	}

	// [7 8 9 10 1 10] => []
	for i := 0; i < 6; i++ {
		_, ok := queue.Dequeue()
		if !ok {
			t.Error("dequeue failed")
		}
	}
	if queue.Len() != 0 {
		t.Error("queue must be empty", queue.Len())
	}

	// [] => []
	rs, ok = queue.Dequeue()
	if ok || rs != nil {
		t.Error("dequeue must failed", rs, ok)
	}
	if queue.Len() != 0 {
		t.Error("invaid queue.Len()", queue.Len())
	}
}
