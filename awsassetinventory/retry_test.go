package awsassetinventory

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"generic error", errors.New("some error"), false},
		{"ThrottlingException", errors.New("ThrottlingException: Rate exceeded"), true},
		{"Throttling in message", errors.New("request failed: Throttling"), true},
		{"Rate exceeded", errors.New("Rate exceeded for API"), true},
		{"RequestLimitExceeded", errors.New("RequestLimitExceeded: too many requests"), true},
		{"TooManyRequestsException", errors.New("TooManyRequestsException"), true},
		{"access denied", errors.New("AccessDeniedException: not authorized"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryable(tt.err); got != tt.want {
				t.Errorf("isRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetry_Success(t *testing.T) {
	callCount := 0
	result, err := retry(context.Background(), 3, func() (string, error) {
		callCount++
		return "success", nil
	})

	if err != nil {
		t.Errorf("retry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("retry() result = %v, want success", result)
	}
	if callCount != 1 {
		t.Errorf("retry() called %d times, want 1", callCount)
	}
}

func TestRetry_RetryableError(t *testing.T) {
	callCount := 0
	throttleErr := errors.New("ThrottlingException: Rate exceeded")

	result, err := retry(context.Background(), 3, func() (string, error) {
		callCount++
		if callCount < 3 {
			return "", throttleErr
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("retry() error = %v, want nil", err)
	}
	if result != "success" {
		t.Errorf("retry() result = %v, want success", result)
	}
	if callCount != 3 {
		t.Errorf("retry() called %d times, want 3", callCount)
	}
}

func TestRetry_NonRetryableError(t *testing.T) {
	callCount := 0
	accessDenied := errors.New("AccessDeniedException: not authorized")

	_, err := retry(context.Background(), 3, func() (string, error) {
		callCount++
		return "", accessDenied
	})

	if err != accessDenied {
		t.Errorf("retry() error = %v, want %v", err, accessDenied)
	}
	if callCount != 1 {
		t.Errorf("retry() called %d times, want 1 (no retry for non-retryable)", callCount)
	}
}

func TestRetry_MaxRetries(t *testing.T) {
	callCount := 0
	throttleErr := errors.New("ThrottlingException: Rate exceeded")

	_, err := retry(context.Background(), 2, func() (string, error) {
		callCount++
		return "", throttleErr
	})

	if err != throttleErr {
		t.Errorf("retry() error = %v, want %v", err, throttleErr)
	}
	// maxRetries=2 means 3 attempts total (initial + 2 retries)
	if callCount != 3 {
		t.Errorf("retry() called %d times, want 3", callCount)
	}
}

func TestRetry_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	callCount := 0
	throttleErr := errors.New("ThrottlingException: Rate exceeded")

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := retry(ctx, 10, func() (string, error) {
		callCount++
		return "", throttleErr
	})

	if !errors.Is(err, context.Canceled) {
		t.Errorf("retry() error = %v, want context.Canceled", err)
	}
	// Should have been cancelled before all retries
	if callCount >= 10 {
		t.Errorf("retry() called %d times, should have been cancelled early", callCount)
	}
}

func TestRetry_ZeroRetries(t *testing.T) {
	callCount := 0
	throttleErr := errors.New("ThrottlingException: Rate exceeded")

	_, err := retry(context.Background(), 0, func() (string, error) {
		callCount++
		return "", throttleErr
	})

	if err != throttleErr {
		t.Errorf("retry() error = %v, want %v", err, throttleErr)
	}
	if callCount != 1 {
		t.Errorf("retry() called %d times, want 1", callCount)
	}
}
