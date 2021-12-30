// Package token contains a standard token bucket implementation.
package token

import (
	"context"
	"fmt"
	"time"
)

// Bucket is an implementation of a standard token Bucket. The Bucket is refilled at some interval
// to a maximum value.
type Bucket struct {
	// tokens represents an full token Bucket at some size. Every entry into the Bucket
	// remove capacity.
	tokens chan struct{}
	// ticker is a ticket that we use to refresh our tokens at some interval.
	ticker *time.Ticker
}

// New creates a Bucket instance. size is how many tokens we can hold. incr is the amount of tokens
// to add at a time. interval is how often to add tokens.
func New(size, incr int, interval time.Duration) (*Bucket, error) {
	if size < 1 {
		return nil, fmt.Errorf("size must be > 1")
	}
	if interval < 1*time.Second {
		return nil, fmt.Errorf("interval must be < 1 second")
	}
	if incr < 1 {
		return nil, fmt.Errorf("incr must be > 0")
	}

	b := Bucket{tokens: make(chan struct{}, size), ticker: time.NewTicker(interval)}
	// This goroutine adds tokens by removing items from our channel. This seems like the
	// opposite logic of what you'd expect, but this is actually an efficient way of implementing
	// a token Bucket usign channels.
	go func() {
		for _ = range b.ticker.C {
			for i := 0; i < incr; i++ {
				select {
				case <-b.tokens:
					continue
				default:
				}
				break
			}
		}
	}()
	return &b, nil
}

// close stops the token Bucket's goroutine. This should be called before throwing away the Bucket.
// If you use the token Bucket after this is called, this can cause major problems like causing the
// token() call to block forever, as there are no more tokens being added.
func (b *Bucket) Close() {
	b.ticker.Stop()
	close(b.tokens)
}

// token blocks until a token is available or the context is cancelled. An error is only returned
// if the context is cancelled.
func (b *Bucket) Token(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case b.tokens <- struct{}{}:
	}
	return nil
}
