package web

import (
	"errors"
	"testing"
	"time"
)

func TestPhoenixBackoff(t *testing.T) {
	cases := []struct {
		failures int
		want     time.Duration
	}{
		{0, 0},
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 16 * time.Second},
		{6, 30 * time.Second},   // 32s capped to 30s
		{7, 30 * time.Second},   // capped
		{100, 30 * time.Second}, // capped, no overflow
	}
	for _, c := range cases {
		if got := phoenixBackoff(c.failures); got != c.want {
			t.Errorf("phoenixBackoff(%d) = %v, want %v", c.failures, got, c.want)
		}
	}
}

// The old listener gave up after a single failed dial. The loop must keep
// retrying instead, with the failure count growing each time it fails.
func TestReconnectLoop_RetriesAfterFailure(t *testing.T) {
	stop := make(chan struct{})
	attempts := 0
	connect := func() error {
		attempts++
		if attempts >= 3 {
			close(stop)
		}
		return errors.New("connection refused")
	}
	var backoffCalls []int
	backoff := func(failures int) time.Duration {
		backoffCalls = append(backoffCalls, failures)
		return 0
	}

	reconnectLoop(stop, connect, backoff)

	if attempts != 3 {
		t.Fatalf("expected 3 connect attempts, got %d", attempts)
	}
	want := []int{1, 2} // 3rd attempt closes stop before its backoff
	if len(backoffCalls) != len(want) {
		t.Fatalf("backoff called %v, want %v", backoffCalls, want)
	}
	for i := range want {
		if backoffCalls[i] != want[i] {
			t.Fatalf("backoff calls = %v, want %v", backoffCalls, want)
		}
	}
}

// A successful connection (connect returns nil) that later drops must reset
// the consecutive-failure count so backoff starts over.
func TestReconnectLoop_ResetsFailureCountAfterSuccess(t *testing.T) {
	stop := make(chan struct{})
	results := []error{errors.New("e"), errors.New("e"), nil, errors.New("e")}
	i := 0
	connect := func() error {
		r := results[i]
		i++
		if i >= len(results) {
			close(stop)
		}
		return r
	}
	var backoffCalls []int
	backoff := func(failures int) time.Duration {
		backoffCalls = append(backoffCalls, failures)
		return 0
	}

	reconnectLoop(stop, connect, backoff)

	want := []int{1, 2, 0} // fail, fail, success(reset); 4th attempt stops before backoff
	if len(backoffCalls) != len(want) {
		t.Fatalf("backoff called %v, want %v", backoffCalls, want)
	}
	for i := range want {
		if backoffCalls[i] != want[i] {
			t.Fatalf("backoff calls = %v, want %v", backoffCalls, want)
		}
	}
}

func TestReconnectLoop_StopsWhenStopClosed(t *testing.T) {
	stop := make(chan struct{})
	close(stop)
	called := false
	connect := func() error { called = true; return nil }
	reconnectLoop(stop, connect, func(int) time.Duration { return 0 })
	if called {
		t.Fatal("connect should not be called when stop is already closed")
	}
}

func TestInterruptibleSleep_ReturnsWhenStopped(t *testing.T) {
	stop := make(chan struct{})
	close(stop)
	done := make(chan struct{})
	go func() {
		interruptibleSleep(time.Hour, stop)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("interruptibleSleep did not return promptly when stop was closed")
	}
}
