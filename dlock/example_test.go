package dlock_test

import (
	"context"
	"fmt"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/dlock"
)

// This example compiles in normal tests. Configure store.mysql before running.
func ExampleLockManager_Acquire() {
	manager, err := dlock.NewLockManagerFromViper()
	if err != nil {
		fmt.Println("initialize:", err)
		return
	}
	defer func() {
		if err := manager.Close(); err != nil {
			fmt.Println("close:", err)
		}
	}()
	ctx := context.Background()
	lease, err := manager.Acquire(ctx, "scheduled-job", 5*time.Second)
	if err != nil {
		fmt.Println("acquire:", err)
		return
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := lease.Release(releaseCtx); err != nil {
			fmt.Println("release:", err)
		}
	}()
	// Renew only the same generation. On ANY error stop business work.
	if err := lease.Renew(ctx, 5*time.Second); err != nil {
		fmt.Println("lost lease:", err)
		return
	}
	workCtx, cancel := context.WithDeadline(ctx, lease.ValidUntil())
	defer cancel()
	if lease.Remaining() == 0 || workCtx.Err() != nil {
		return
	}
	// Pass workCtx to a bounded business operation here. Cancellation is
	// cooperative; strict stale-write rejection requires downstream fencing.
}
