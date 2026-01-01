package helpers

import (
	"testing"
	"time"
)

// WaitConfig configures wait behavior.
type WaitConfig struct {
	Timeout  time.Duration
	Interval time.Duration
}

// DefaultWait is the default wait configuration.
var DefaultWait = WaitConfig{
	Timeout:  3 * time.Second,
	Interval: 20 * time.Millisecond,
}

// Wait polls a condition until it returns true or times out.
func Wait(t *testing.T, condition func() bool, message string) {
	t.Helper()
	WaitWithConfig(t, DefaultWait, condition, message)
}

// WaitWithConfig polls with custom configuration.
func WaitWithConfig(t *testing.T, cfg WaitConfig, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(cfg.Timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(cfg.Interval)
	}

	t.Fatalf("timeout waiting for: %s", message)
}

// WaitFor is a generic version that returns a value when condition is met.
func WaitFor[T any](t *testing.T, getter func() (T, bool), message string) T {
	t.Helper()
	return WaitForWithConfig(t, DefaultWait, getter, message)
}

// WaitForWithConfig is WaitFor with custom configuration.
func WaitForWithConfig[T any](t *testing.T, cfg WaitConfig, getter func() (T, bool), message string) T {
	t.Helper()

	deadline := time.Now().Add(cfg.Timeout)
	for time.Now().Before(deadline) {
		if val, ok := getter(); ok {
			return val
		}
		time.Sleep(cfg.Interval)
	}

	t.Fatalf("timeout waiting for: %s", message)
	var zero T
	return zero
}

// Sleep is a wrapper around time.Sleep for readability in tests.
func Sleep(d time.Duration) {
	time.Sleep(d)
}

// Retry retries a function until it succeeds or max attempts are reached.
func Retry(t *testing.T, maxAttempts int, interval time.Duration, fn func() error) {
	t.Helper()

	var lastErr error
	for i := range maxAttempts {
		if err := fn(); err == nil {
			return
		} else {
			lastErr = err
			if i < maxAttempts-1 {
				time.Sleep(interval)
			}
		}
	}

	t.Fatalf("failed after %d attempts: %v", maxAttempts, lastErr)
}

// WaitWithBackoff polls with exponential backoff until condition returns true or timeout.
// Starts with initialInterval and doubles each attempt up to maxInterval.
func WaitWithBackoff(t *testing.T, timeout time.Duration, initialInterval, maxInterval time.Duration, condition func() bool, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	interval := initialInterval

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
		if interval < maxInterval {
			interval = min(interval*2, maxInterval)
		}
	}

	t.Fatalf("timeout waiting for: %s", message)
}

// WaitForWithBackoff is a generic version with exponential backoff that returns a value.
func WaitForWithBackoff[T any](t *testing.T, timeout time.Duration, initialInterval, maxInterval time.Duration, getter func() (T, bool), message string) T {
	t.Helper()

	deadline := time.Now().Add(timeout)
	interval := initialInterval

	for time.Now().Before(deadline) {
		if val, ok := getter(); ok {
			return val
		}
		time.Sleep(interval)
		if interval < maxInterval {
			interval = min(interval*2, maxInterval)
		}
	}

	t.Fatalf("timeout waiting for: %s", message)
	var zero T
	return zero
}
