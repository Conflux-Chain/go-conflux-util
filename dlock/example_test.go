package dlock_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/dlock"
)

func ExampleLockManager_Acquire() {
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
	if err := lockMan.Acquire(ctx, intent); err != nil {
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

	// Perform bounded, cancellable work while the lease is valid.
	// Renew before expiry with the same intent; stop work if renewal fails.
	if err := lockMan.Acquire(ctx, intent); err != nil {
		log.Print(err)
		return
	}
}
