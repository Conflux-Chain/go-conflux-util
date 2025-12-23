# Channel Utilities

This package provides more powerful channel than the built-in one.

## Memory Bounded Channel

This is a built-in channel alike structure that bounds both the number of items and the total memory size used. It aims to prevent OOM if each item is very large in channel. See examples below:

```go
func Run(ctx context.Context) {
    var wg sync.WaitGroup

    dataCh := channel.NewMemoryBoundedChannel[Foo](1024, 256_000_000)
    defer dataCh.Close()

    wg.Add(1)
    go produce(ctx, &wg, dataCh.SendCh())

    wg.Add(1)
    go consume(ctx, &wg, dataCh.RecvCh())

    wg.Wait()
}
```
