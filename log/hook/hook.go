package hook

import (
	"context"
	"sync"

	"github.com/Conflux-Chain/go-conflux-util/alert"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// AddAlertHook attaches a custom logrus Hook for generating alert notifications
// based on configured levels and channels.
// It supports both synchronous and asynchronous operation modes, with optional
// graceful shutdown integration.
func AddAlertHook(ctx context.Context, wg *sync.WaitGroup, conf AlertConfig) error {
	if len(conf.Channels) == 0 {
		// No channels configured, so no hook needs to be added.
		return nil
	}

	// Retrieve and validate configured alert channels.
	var chs []alert.Channel
	for _, chn := range conf.Channels {
		ch, ok := alert.DefaultManager().Channel(chn)
		if !ok {
			return alert.ErrChannelNotFound(chn)
		}
		chs = append(chs, ch)
	}

	// Parse the configured log level for alert triggering.
	lvl, err := logrus.ParseLevel(conf.Level)
	if err != nil {
		return errors.WithMessage(err, "failed to parse log level")
	}

	var hookLvls []logrus.Level
	for l := logrus.PanicLevel; l <= lvl; l++ {
		hookLvls = append(hookLvls, l)
	}

	// Instantiate the base AlertHook.
	var alertHook logrus.Hook = NewAlertHook(hookLvls, chs, conf.SendTimeout)

	// Wrap with asynchronous processing if configured.
	if conf.Async.NumWorkers > 0 {
		alertHook = wrapAsyncHook(ctx, wg, alertHook, conf.Async)
	}

	// Finally, add the hook to Logrus.
	logrus.AddHook(alertHook)

	return nil
}

// AddFileHook attaches a custom logrus Hook for writing to log files based on configured levels and rotation settings.
// Supports synchronous and asynchronous modes with optional graceful shutdown integration.
func AddFileHook(ctx context.Context, wg *sync.WaitGroup, formatter logrus.Formatter, conf FileConfig) error {
	// Return early if no groups are configured.
	if len(conf.Groups) == 0 {
		return nil
	}

	// Create a file hook using the provided configuration and formatter.
	var fileHook logrus.Hook
	fileHook, err := NewFileHook(conf, formatter)
	if err != nil {
		return errors.Wrap(err, "failed to create file hook")
	}

	// Wrap the hook with asynchronous processing if configured.
	if conf.Async.NumWorkers > 0 {
		fileHook = wrapAsyncHook(ctx, wg, fileHook, conf.Async)
	}

	// Attach the hook to logrus.
	logrus.AddHook(fileHook)

	return nil
}

// wrapAsyncHook wraps the given hook with asynchronous processing, optionally integrating
// graceful shutdown support if a context and wait group are provided.
func wrapAsyncHook(
	ctx context.Context, wg *sync.WaitGroup, hook logrus.Hook, opt AsyncOption) *AsyncHook {
	if ctx != nil && wg != nil {
		return NewAsyncHookWithCtx(ctx, wg, hook, opt)
	}

	return NewAsyncHook(hook, opt)
}
