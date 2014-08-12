package hydra

import (
	"container/list"
	"github.com/fujiwara/fluent-agent-hydra/fluent"
)

type MessageQueue struct {
	list        *list.List
	locker      chan interface{}
	messages    int64
	maxMessages int64
}

func NewMessageQueue(maxMessages int) *MessageQueue {
	locker := make(chan interface{}, 1)
	q := &MessageQueue{
		list:        list.New(),
		locker:      locker,
		messages:    0,
		maxMessages: int64(maxMessages),
	}
	q.unlock()
	return q
}

func (q *MessageQueue) lock() {
	<-q.locker
}

func (q *MessageQueue) unlock() {
	q.locker <- nil
}

func (q *MessageQueue) Enqueue(recordSet *fluent.FluentRecordSet) int64 {
	q.lock()
	defer q.unlock()
	messages := int64(len(recordSet.Records))
	disposed := int64(0)
	for q.messages+messages > q.maxMessages && q.list.Len() > 0 {
		_rs := q.dequeue() // dispose first value
		rs := _rs.(*fluent.FluentRecordSet)
		disposed += int64(len(rs.Records))
		q.messages -= int64(len(rs.Records))
	}
	q.list.PushBack(recordSet)
	q.messages += messages
	return disposed
}

func (q *MessageQueue) Dequeue() (*fluent.FluentRecordSet, bool) {
	q.lock()
	defer q.unlock()
	if q.list.Len() == 0 {
		q.messages = 0
		return nil, false
	}
	_rs := q.dequeue()
	rs := _rs.(*fluent.FluentRecordSet)
	q.messages -= int64(len(rs.Records))
	return rs, true
}

func (q *MessageQueue) dequeue() interface{} {
	return q.list.Remove(q.list.Front())
}

func (q *MessageQueue) Len() int {
	q.lock()
	defer q.unlock()
	return int(q.messages)
}
