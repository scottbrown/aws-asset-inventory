package awsassetinventory

import (
	"context"
	"math/rand"
	"strings"
	"time"
)

const (
	DefaultMaxRetries     = 3
	DefaultBaseDelay      = 100 * time.Millisecond
	DefaultMaxDelay       = 5 * time.Second
	DefaultMaxConcurrency = 5
)

// isRetryable checks if an error is retryable (throttling, transient).
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "ThrottlingException") ||
		strings.Contains(msg, "Throttling") ||
		strings.Contains(msg, "Rate exceeded") ||
		strings.Contains(msg, "RequestLimitExceeded") ||
		strings.Contains(msg, "TooManyRequestsException")
}

// retry executes fn with exponential backoff for retryable errors.
func retry[T any](ctx context.Context, maxRetries int, fn func() (T, error)) (T, error) {
	var result T
	var err error
	delay := DefaultBaseDelay

	for attempt := 0; attempt <= maxRetries; attempt++ {
		result, err = fn()
		if err == nil {
			return result, nil
		}

		if !isRetryable(err) || attempt == maxRetries {
			return result, err
		}

		// Add jitter: 50-150% of delay
		jitter := time.Duration(rand.Int63n(int64(delay)))
		sleep := delay + jitter/2

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(sleep):
		}

		// Exponential backoff with cap
		delay *= 2
		if delay > DefaultMaxDelay {
			delay = DefaultMaxDelay
		}
	}

	return result, err
}
