package report

import "sync"

// RingBuff 环形队列
type RingBuff[T any] struct {
	items    []T
	size     int
	capacity int
	head     int
	tail     int
	mutex    sync.Mutex
}

func NewRingBuff[T any](capacity int) *RingBuff[T] {
	return &RingBuff[T]{
		items:    make([]T, capacity),
		size:     0,
		capacity: capacity,
		head:     0,
		tail:     0,
	}
}

func (rb *RingBuff[T]) Enqueue(val T) bool {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	if rb.size == rb.capacity {
		return false // 队列已满
	}
	rb.items[rb.tail] = val
	rb.tail = (rb.tail + 1) % rb.capacity
	rb.size++
	return true
}

func (rb *RingBuff[T]) Dequeue() (T, bool) {
	rb.mutex.Lock()
	defer rb.mutex.Unlock()
	var t T
	if rb.size == 0 {
		return t, false // 队列为空
	}
	val := rb.items[rb.head]
	rb.head = (rb.head + 1) % rb.capacity
	rb.size--
	return val, true
}
func (rb *RingBuff[T]) Current() T {
	var t T
	if rb.size == 0 {
		return t // 队列为空
	}
	return rb.items[rb.head]
}
func (rb *RingBuff[T]) Size() int {
	return rb.size
}

func (rb *RingBuff[T]) Surplus() []T {
	var result []T
	size := rb.size
	head := rb.head
	for index := 0; index < size; index++ {
		result = append(result, rb.items[(index+head)%rb.capacity])
	}
	return result
}
func (rb *RingBuff[T]) SurplusCount() int {
	return rb.capacity - rb.size
}
