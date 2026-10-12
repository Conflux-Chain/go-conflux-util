# Channel Utilities

This package provides more powerful channel than the built-in one.

## Memory Bounded Channel

This is a built-in channel alike structure that bounds both the number of items and the total memory size used. It aims to prevent OOM if each item is very large in channel.

Like a built-in buffered channel, `Close()` preserves accepted items for receiving. Stop all senders before calling `Close()`, and use this method instead of closing `SendCh()` directly. Continue receiving until `RecvCh()` closes.

`Close()` returns without waiting for the buffer to drain or the background goroutines to exit. If consumers exit early, the caller must arrange to drain the remaining items. Otherwise, a blocked forwarding goroutine keeps the channel and its buffered data alive even after the caller drops its reference.

The example below runs a producer and a consumer, closes the channel after production finishes, and waits for consumption to complete. Both functions call `Done()` on their supplied wait group when they exit.

On cancellation, `produce` must stop sending, while `consume` must keep draining until the receiving channel closes, even if it stops processing items.

```go
func Run(ctx context.Context) {
    dataCh := channel.NewMemoryBoundedChannel[Foo](1024, 256_000_000)

    var producers, consumers sync.WaitGroup
    producers.Add(1)
    go produce(ctx, &producers, dataCh.SendCh())

    consumers.Add(1)
    go consume(ctx, &consumers, dataCh.RecvCh())

    producers.Wait()
    dataCh.Close()
    consumers.Wait()
}
```
