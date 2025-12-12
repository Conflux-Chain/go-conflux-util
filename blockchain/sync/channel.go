package sync

import (
	"container/list"
	"sync"
)

// Sizable is the interface implemented by types that support to compute memory size.
type Sizable interface {
	Size() int // Memory size in bytes
}

// MemoryBoundedChannel is a channel-like structure that bounds both the number of items and
// the total memory size of the items in the channel.
type MemoryBoundedChannel[T Sizable] struct {
	mu           sync.Mutex
	notFullCond  *sync.Cond // to wake up goroutine to write buffer
	notEmptyCond *sync.Cond // to wake up goroutine to read buffer

	buffer *list.List // underlying buffer

	capacity int // max number of items in channel
	maxBytes int // max total memory size in channel
	curBytes int // current total memory size in channel

	sendCh chan T // channel for sending data
	recvCh chan T // channel for receiving data

	closed bool // indicates whether the channel is closed
}

// NewMemoryBoundedChannel creates a new MemoryBoundedChannel with the specified capacity and maximum memory size.
//
// It will panic if capacity or maxBytes <= 0.
//
// To provide native channel alike behavior, this channel has to receive the data before checking the memory size.
// As a result, the actual maximum memory size is maxBytes plus the last data size.
func NewMemoryBoundedChannel[T Sizable](capacity, maxBytes int) *MemoryBoundedChannel[T] {
	return newMemoryBoundedChannelWithStopCh[T](capacity, maxBytes, nil, nil)
}

func newMemoryBoundedChannelWithStopCh[T Sizable](capacity, maxBytes int, sendStopCh, recvStopCh chan<- struct{}) *MemoryBoundedChannel[T] {
	if capacity <= 0 || maxBytes <= 0 {
		panic("capacity or maxBytes <= 0")
	}

	channel := MemoryBoundedChannel[T]{
		buffer:   list.New(),
		capacity: capacity,
		maxBytes: maxBytes,
		sendCh:   make(chan T),
		recvCh:   make(chan T),
	}

	channel.notFullCond = sync.NewCond(&channel.mu)
	channel.notEmptyCond = sync.NewCond(&channel.mu)

	go channel.loopEnqueue(sendStopCh)
	go channel.loopDequeue(recvStopCh)

	return &channel
}

// SendCh returns the sending channel.
func (ch *MemoryBoundedChannel[T]) SendCh() chan<- T {
	return ch.sendCh
}

// RecvCh returns the receiving channel.
func (ch *MemoryBoundedChannel[T]) RecvCh() <-chan T {
	return ch.recvCh
}

// Close releases resources associated with the channel.
//
// Note, it will panic if continue to write data after closed, which is the same as the built-in channel behavior.
func (ch *MemoryBoundedChannel[T]) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// already closed
	if ch.closed {
		return
	}

	ch.closed = true

	// Close sendCh to stop loopEnqueue goroutine.
	// Note, recvCh cannot be closed here, otherwise loopDequeue
	// goroutine may panic to write data into recvCh.
	close(ch.sendCh)

	// wake up all waiting goroutines
	ch.notFullCond.Broadcast()
	ch.notEmptyCond.Broadcast()
}

func (ch *MemoryBoundedChannel[T]) loopEnqueue(stopCh chan<- struct{}) {
	for item := range ch.sendCh {
		ch.enqueue(item)
	}

	if stopCh != nil {
		stopCh <- struct{}{}
	}
}

func (ch *MemoryBoundedChannel[T]) enqueue(item T) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// add item to buffer
	ch.buffer.PushBack(item)
	ch.curBytes += item.Size()

	// broadcast to waiting receivers
	ch.notEmptyCond.Broadcast()

	// blocking sending channel if buffer is full and this channel not closed yet
	for !ch.closed && (ch.buffer.Len() >= ch.capacity || ch.curBytes >= ch.maxBytes) {
		ch.notFullCond.Wait()
	}
}

func (ch *MemoryBoundedChannel[T]) loopDequeue(stopCh chan<- struct{}) {
	for {
		val, ok := ch.peek()
		if !ok {
			// channel is closed and buffer is empty
			break
		}

		ch.recvCh <- val

		ch.dequeue()
	}

	close(ch.recvCh)

	if stopCh != nil {
		stopCh <- struct{}{}
	}
}

func (ch *MemoryBoundedChannel[T]) peek() (val T, ok bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// wait until there is data
	for !ch.closed && ch.buffer.Len() == 0 {
		ch.notEmptyCond.Wait()
	}

	front := ch.buffer.Front()

	// channel is closed and buffer is empty
	if front == nil {
		return
	}

	return front.Value.(T), true
}

func (ch *MemoryBoundedChannel[T]) dequeue() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// remove the front element, which must be exist
	front := ch.buffer.Front()
	ch.buffer.Remove(front)
	ch.curBytes -= front.Value.(T).Size()

	// broadcast to waiting senders
	ch.notFullCond.Broadcast()
}
