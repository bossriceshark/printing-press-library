// Copyright 2026 Cursor Agent. Licensed under Apache-2.0. See LICENSE.
package client

import (
	"context"
	"sync"
	"time"
)

// Limiter is a token bucket. New Central's account-wide cap is 10 requests/second,
// shared with every other integration on the tenant, so the default is 10.
type Limiter struct {
	mu     sync.Mutex
	rate   float64
	burst  float64
	tokens float64
	last   time.Time
	now    func() time.Time
	sleep  func(time.Duration)
	// fakeClock is set when tests inject now/sleep. Production uses the real sleeper.
	fakeClock bool
}

func NewLimiter(rate float64) *Limiter {
	if rate <= 0 {
		rate = 10
	}
	return &Limiter{
		rate:   rate,
		burst:  rate,
		tokens: rate,
		now:    time.Now,
		sleep:  time.Sleep,
	}
}

// Wait blocks until one request token is available or ctx is cancelled.
func (l *Limiter) Wait(ctx context.Context) error {
	if l == nil {
		return nil
	}
	for {
		l.mu.Lock()
		now := l.now()
		if l.last.IsZero() {
			l.last = now
		}
		elapsed := now.Sub(l.last).Seconds()
		if elapsed > 0 {
			l.tokens += elapsed * l.rate
			if l.tokens > l.burst {
				l.tokens = l.burst
			}
			l.last = now
		}
		if l.tokens >= 1 {
			l.tokens--
			l.mu.Unlock()
			return nil
		}
		need := (1 - l.tokens) / l.rate
		l.mu.Unlock()
		d := time.Duration(need * float64(time.Second))
		if d < time.Millisecond {
			d = time.Millisecond
		}
		if err := l.wait(ctx, d); err != nil {
			return err
		}
	}
}

func (l *Limiter) wait(ctx context.Context, d time.Duration) error {
	if l.fakeClock {
		l.sleep(d)
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
