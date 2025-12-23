package channel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type SizedData struct {
	id   int
	size int
}

func (data SizedData) Size() int {
	return data.size
}

func TestMemoryBoundedChannelCapacity(t *testing.T) {
	testMemoryBoundedChannelCapacity(t, 1)
	testMemoryBoundedChannelCapacity(t, 3)
	testMemoryBoundedChannelCapacity(t, 100)
}

func testMemoryBoundedChannelCapacity(t *testing.T, capacity int) {
	assert.Greater(t, capacity, 0)

	channel := NewMemoryBoundedChannel[SizedData](capacity, 100*capacity)

	// write data
	for i := 0; i < capacity; i++ {
		mustWriteChannel(channel.SendCh(), SizedData{i, 10})
	}

	// cannot write data again, since channel is waiting for notFullCond
	mustWriteChannelBlocked(channel.SendCh(), SizedData{capacity, 10})

	// read data
	for i := 0; i < capacity; i++ {
		assertReadChannel(t, SizedData{i, 10}, channel.RecvCh())
	}
}

func mustWriteChannel[T any](ch chan<- T, data T) {
	select {
	case ch <- data:
		return
	case <-time.After(5 * time.Second):
		panic("Timeout to write data into channel")
	}
}

func mustWriteChannelBlocked[T any](ch chan<- T, data T) {
	select {
	case ch <- data:
		panic("Unexpected to write data into channel")
	case <-time.After(300 * time.Millisecond):
		return
	}
}

func assertReadChannel[T any](t *testing.T, expected T, ch <-chan T) {
	select {
	case actual := <-ch:
		assert.Equal(t, expected, actual)
	case <-time.After(5 * time.Second):
		panic("Timeout to read data from channel")
	}
}

func TestMemoryBoundedChannelMaxBytes(t *testing.T) {
	channel := NewMemoryBoundedChannel[SizedData](3, 100)

	// maxBytes readched
	mustWriteChannel(channel.SendCh(), SizedData{1, 100})
	// cannot write anymore, since maxBytes reached
	mustWriteChannelBlocked(channel.SendCh(), SizedData{2, 1})

	//read data
	assertReadChannel(t, SizedData{1, 100}, channel.RecvCh())

	// could exceed maxBytes for the last data
	mustWriteChannel(channel.SendCh(), SizedData{3, 99})
	mustWriteChannel(channel.SendCh(), SizedData{4, 10})
	mustWriteChannelBlocked(channel.SendCh(), SizedData{5, 10})
	assertReadChannel(t, SizedData{3, 99}, channel.RecvCh())
	assertReadChannel(t, SizedData{4, 10}, channel.RecvCh())
}

func TestMemoryBoundedChannelClosed(t *testing.T) {
	sendStopCh := make(chan struct{}, 1)
	defer close(sendStopCh)
	recvStopCh := make(chan struct{}, 1)
	defer close(recvStopCh)

	channel := newMemoryBoundedChannelWithStopCh[SizedData](3, 100, sendStopCh, recvStopCh)

	// write data
	mustWriteChannel(channel.SendCh(), SizedData{1, 10})
	mustWriteChannel(channel.SendCh(), SizedData{2, 10})
	mustWriteChannel(channel.SendCh(), SizedData{3, 10})

	// sending channel is blocked and waiting on notFullCond
	mustWriteChannelBlocked(channel.SendCh(), SizedData{4, 10})

	// close channel
	channel.Close()

	// no effect if close again
	channel.Close()

	// sending goroutine waken up and completed
	assertReadChannel(t, struct{}{}, sendStopCh)

	// continue to read data from channel even closed already
	// this is the same behavior of golang built-in channel
	assertReadChannel(t, SizedData{1, 10}, channel.RecvCh())
	assertReadChannel(t, SizedData{2, 10}, channel.RecvCh())
	assertReadChannel(t, SizedData{3, 10}, channel.RecvCh())

	// receiving goroutine waken up and completed
	assertReadChannel(t, struct{}{}, recvStopCh)
}
