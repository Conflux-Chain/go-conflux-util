package channel

import (
	"container/list"
	"sync"
	"sync/atomic"
)

// Sizable is the interface implemented by types that support to compute memory size.
type Sizable interface {
	// Size returns a non-negative estimate of the item's memory size in bytes.
	Size() int
}

type Sized[T any] struct {
	Data T
	Size int
}

// MemoryBoundedChannel is a channel-like structure that bounds both the number of items and
// the total estimated memory size of the items in the channel.
//
// Each item's Size is recorded once when enqueued; later changes to the item do not
// update the recorded size. The sum of recorded sizes in the buffer must fit in an int.
// The byte limit uses these estimates, not the process's actual memory usage.
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

	sendClosed atomic.Bool // indicates whether the send channel is closed
	recvClosed atomic.Bool // indicates whether the recv channel is closed
}

// NewMemoryBoundedChannel creates a new MemoryBoundedChannel with the specified capacity and maximum memory size.
//
// It will panic if capacity or maxBytes <= 0.
//
// To provide native channel alike behavior, this channel has to receive the data before checking the memory size.
// As a result, the total recorded size can exceed maxBytes by up to the last item's Size.
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
// Use Close to close it after all senders have stopped.
func (ch *MemoryBoundedChannel[T]) SendCh() chan<- T {
	return ch.sendCh
}

// RecvCh returns the receiving channel.
// It closes after Close has been called and all accepted items have been received.
func (ch *MemoryBoundedChannel[T]) RecvCh() <-chan T {
	return ch.recvCh
}

// Close closes the sending channel and preserves all accepted items for receiving.
// Repeated calls have no effect.
//
// Callers must stop all senders before calling Close. Sending after Close panics,
// as with a built-in channel.
//
// Close does not wait for the buffer to drain or the background goroutines to exit.
// Callers must keep receiving until RecvCh closes to allow resources to be released.
// If consumers stop early, arrange for the remaining items to be drained; otherwise
// a blocked forwarding goroutine retains the channel and its buffered data even if
// the caller drops its reference to the channel.
func (ch *MemoryBoundedChannel[T]) Close() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// already closed
	if !ch.sendClosed.CompareAndSwap(false, true) {
		return
	}

	// Close sendCh to stop loopEnqueue goroutine.
	// Note, recvCh cannot be closed here, otherwise loopDequeue
	// goroutine may panic to write data into recvCh.
	close(ch.sendCh)

	// wake up loopEnqueue goroutine
	ch.notFullCond.Broadcast()
}

func (ch *MemoryBoundedChannel[T]) loopEnqueue(stopCh chan<- struct{}) {
	for item := range ch.sendCh {
		ch.enqueue(item)
	}

	// Hold the same lock as peek so closure cannot be missed before Wait.
	ch.mu.Lock()
	ch.recvClosed.Store(true)
	ch.notEmptyCond.Broadcast()
	ch.mu.Unlock()

	if stopCh != nil {
		stopCh <- struct{}{}
	}
}

func (ch *MemoryBoundedChannel[T]) enqueue(item T) {
	data := Sized[T]{
		Data: item,
		Size: item.Size(),
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	// add item to buffer
	ch.buffer.PushBack(data)
	ch.curBytes += data.Size

	// broadcast to waiting receivers
	ch.notEmptyCond.Broadcast()

	// blocking sending channel if buffer is full and this channel not closed yet
	for !ch.sendClosed.Load() && (ch.buffer.Len() >= ch.capacity || ch.curBytes >= ch.maxBytes) {
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

		ch.recvCh <- val.Data

		ch.dequeue()
	}

	close(ch.recvCh)

	if stopCh != nil {
		stopCh <- struct{}{}
	}
}

func (ch *MemoryBoundedChannel[T]) peek() (val Sized[T], ok bool) {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// wait until there is data
	for !ch.recvClosed.Load() && ch.buffer.Len() == 0 {
		ch.notEmptyCond.Wait()
	}

	front := ch.buffer.Front()

	// channel is closed and buffer is empty
	if front == nil {
		return
	}

	return front.Value.(Sized[T]), true
}

func (ch *MemoryBoundedChannel[T]) dequeue() {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// remove the front element, which must be exist
	front := ch.buffer.Front()
	ch.buffer.Remove(front)
	ch.curBytes -= front.Value.(Sized[T]).Size

	// broadcast to waiting senders
	ch.notFullCond.Broadcast()
}
