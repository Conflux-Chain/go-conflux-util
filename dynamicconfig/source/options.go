package source

import (
	"context"

	"go-micro.dev/v4/config/source"
)

// WithContext sets the context.
func WithContext(ctx context.Context) source.Option {
	return func(o *source.Options) {
		o.Context = ctx
	}
}
