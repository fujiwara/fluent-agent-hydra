package hydra_test

import (
	"fmt"
	"github.com/fujiwara/fluent-agent-hydra/hydra"
	"log"
	"testing"
	"time"
)

func TestMessageQueue(t *testing.T) {
	length := 10
	queue := hydra.NewMessageQueue(length, false)
	for i := 0; i < length; i++ {
		v := fmt.Sprintf("message%d", i)
		err := queue.Enqueue(v)
		if err != nil {
			t.Error(err)
		}
		if queue.Len() != i+1 {
			t.Error("invalid queue.Len()", queue)
		}
	}
	err := queue.Enqueue("must be overflow")
	if err == nil {
		t.Error("not blocked enqueu", queue)
	}
	v, ok := queue.Dequeue()
	if !ok {
		t.Error("dequeue failed", queue)
	}
	if v.(string) != "message0" {
		t.Error("invalid dequeued value", v)
	}
}

func TestMessageQueueMultiThreaded(t *testing.T) {
	threads := 4
	n := 100
	queue := hydra.NewMessageQueue(n * threads * 2, false)
	done := make(chan int)

	for i := 0; i < threads; i++ {
		go testDoEnqueue(t, queue, n)
		go testDoDequeue(t, queue, n, done)
	}
	dequeued := 0
DEQUEUE:
	for i := 0; i < threads; i++ {
		select {
		case <-time.After(1 * time.Second):
			break DEQUEUE
		case x := <-done:
			log.Println("dequeued", x)
			dequeued += x
		}
	}
	if dequeued != n * threads {
		t.Errorf("enqueued", n, "dequeued", dequeued)
	}
}

func testDoEnqueue(t *testing.T, queue *hydra.MessageQueue, n int) {
	for i := 0; i < n; i++ {
		err := queue.Enqueue(i)
		if err != nil {
			t.Error("enqueue failed", err)
		}
	}
	log.Println("enqueued", n)
}

func testDoDequeue(t *testing.T, queue *hydra.MessageQueue, n int, done chan int) {
	dequeued := 0
	for dequeued < n {
		_, ok := queue.Dequeue()
		if ok {
			dequeued++
		}
	}
	done <- dequeued
}


func TestMessageQueueDispose(t *testing.T) {
	length := 10
	queue := hydra.NewMessageQueue(length, true)
	for i := 0; i < length; i++ {
		v := fmt.Sprintf("message%d", i)
		err := queue.Enqueue(v)
		if err != nil {
			t.Error(err)
		}
		if queue.Len() != i+1 {
			t.Error("invalid queue.Len()", queue)
		}
	}
	err := queue.Enqueue("must be disposed first")
	if err != nil {
		t.Error("queue is full?", err)
	}
	if queue.Len() != length {
		t.Error("invalid queue length", queue.Len())
	}
	if queue.Disposed() != 1 {
		t.Error("invalid queue disposed", queue.Disposed())
	}
	v, ok := queue.Dequeue()
	if !ok {
		t.Error("dequeue failed", queue)
	}
	if v.(string) != "message1" {
		t.Error("invalid dequeued value", v)
	}
}
