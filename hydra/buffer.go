package hydra

import (
	"container/list"
	"errors"
)

type MessageQueue struct {
	list      *list.List
	locker    chan int
	maxLength int
	drain     bool
	disposed  int
}

func NewMessageQueue(maxLength int, drain bool) *MessageQueue {
	ch := make(chan int, 1)
	q := &MessageQueue{
		list:      list.New(),
		locker:    ch,
		maxLength: maxLength,
		drain:     drain,
	}
	q.unlock()
	return q
}

func (q *MessageQueue) lock() {
	<-q.locker
}

func (q *MessageQueue) unlock() {
	q.locker <- 1
}

func (q *MessageQueue) Enqueue(value interface{}) error {
	q.lock()
	defer q.unlock()
	if q.list.Len() >= q.maxLength {
		if q.drain {
			q.dequeue() // dispose first value
			q.disposed++
		} else {
			return errors.New("queue is full")
		}
	}
	q.list.PushBack(value)
	return nil
}

func (q *MessageQueue) Dequeue() (interface{}, bool) {
	q.lock()
	defer q.unlock()
	if q.list.Len() == 0 {
		return nil, false
	}
	value := q.dequeue()
	return value, true
}

func (q *MessageQueue) dequeue() interface{} {
	return q.list.Remove(q.list.Front())
}

func (q *MessageQueue) Len() int {
	q.lock()
	defer q.unlock()
	return q.list.Len()
}

func (q *MessageQueue) Disposed() int {
	q.lock()
	defer q.unlock()
	return q.disposed
}
