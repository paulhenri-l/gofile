package chans

import (
	"context"
	"time"
)

// Read from the given channel of ItemType as long as the ctx is not done.
func OrDoneTimeTime(ctx context.Context, input <-chan time.Time) <-chan time.Time {
	out := make(chan time.Time)

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case item, ok := <-input:
				if ok != true {
					return
				}

				out <- item
			}
		}
	}()

	return out
}
