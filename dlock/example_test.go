package dlock_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/Conflux-Chain/go-conflux-util/dlock"
	"github.com/Conflux-Chain/go-conflux-util/store"
)

func ExampleLockManager_AcquireWithDeadline() {
	// The application initializes the database and owns its connection pool.
	config := store.MustNewConfigFromViper()
	db, err := config.OpenOrCreate(&dlock.Dlock{})
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		if err := store.NewStore(db).Close(); err != nil {
			log.Printf("close database: %v", err)
		}
	}()

	ctx := context.Background()
	backend := dlock.NewMySQLBackend(db)
	manager := dlock.NewLockManager(backend)

	var nonce [16]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		log.Print(err)
		return
	}
	intent := dlock.NewLockIntent("dlock_key", fmt.Sprintf("%x", nonce), 15*time.Second)
	deadline, err := manager.AcquireWithDeadline(ctx, intent)
	if err != nil {
		log.Print(err)
		return
	}
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := manager.Release(releaseCtx, intent); err != nil {
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
	deadline, err = manager.AcquireWithDeadline(ctx, intent)
	if err != nil {
		log.Print(err)
		return
	}
	cancelWork()
	workCtx, cancelWork = context.WithDeadline(ctx, deadline)
	defer cancelWork()
	if err := workCtx.Err(); err != nil {
		log.Print(err)
		return
	}
	// Continue protected work using the renewed workCtx.
}
