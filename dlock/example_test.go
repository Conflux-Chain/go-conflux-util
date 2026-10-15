package dlock_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/dlock"
)

func ExampleLockManager_AcquireWithDeadline() {
	lockMan, err := dlock.OpenLockManagerFromViper()
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		if err := lockMan.Close(); err != nil {
			log.Printf("close lock manager: %v", err)
		}
	}()

	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		log.Print(err)
		return
	}
	ctx := context.Background()
	intent := dlock.NewLockIntent("dlock_key", fmt.Sprintf("%x", nonce), 15*time.Second)
	deadline, err := lockMan.AcquireWithDeadline(ctx, intent)
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := lockMan.Release(releaseCtx, intent); err != nil {
			log.Printf("release lock: %v", err)
		}
	}()

	workCtx, cancelWork := context.WithDeadline(ctx, deadline)
	defer cancelWork()
	// Pass workCtx to protected operations and stop by deadline.
	if err := workCtx.Err(); err != nil {
		log.Print(err)
		return
	}

	// Renew before expiry using the original context and the same intent.
	deadline, err = lockMan.AcquireWithDeadline(ctx, intent)
	if err != nil {
		log.Print(err)
		return
	}
	// Cancel the old work context and create a new one from ctx and deadline
	// before continuing work under the renewed lease.
}
