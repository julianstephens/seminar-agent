package sse_test

import (
	"log/slog"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/julianstephens/formation/internal/sse"
)

// newTestHub starts an in-process miniredis instance and returns a Hub wired to it.
func newTestHub(t *testing.T) *sse.Hub {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	return sse.New(slog.Default(), rdb, "test:")
}

// TestSlowSubscriberEviction verifies that a subscriber whose channel fills up
// is automatically evicted after maxConsecutiveDrops failed sends, and that its
// channel is closed so the consumer goroutine detects the eviction.
func TestSlowSubscriberEviction(t *testing.T) {
	t.Parallel()

	hub := newTestHub(t)
	const sessionID = "session-slow"

	// Subscribe but never read from the channel (simulates a slow consumer).
	ch, unsub, err := hub.Subscribe(sessionID, "slow-owner")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	// unsub is idempotent; safe to call even if the subscriber was already evicted.
	defer unsub()

	event := sse.Event{Type: sse.EventError, Data: sse.ErrorPayload{Message: "test"}}

	// Fill the buffer (256) plus 3 more sends to trigger the drop threshold.
	const overflowCount = 256 + 10
	for i := 0; i < overflowCount; i++ {
		hub.Publish(sessionID, event)
	}

	// The subscriber channel must be closed by the eviction logic.
	// Drain any buffered events and then wait for the channel close.
	deadline := time.After(3 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				// Channel was closed: eviction confirmed.
				return
			}
		case <-deadline:
			t.Fatal("slow subscriber channel was not closed within the timeout")
		}
	}
}

// TestTooManySubscribersReturnsError confirms that Subscribe returns
// ErrTooManySubscribers once the per-session cap is reached.
func TestTooManySubscribersReturnsError(t *testing.T) {
	t.Parallel()

	hub := newTestHub(t)
	const sessionID = "session-cap"
	const cap = 100

	unsubs := make([]func(), 0, cap)
	for i := 0; i < cap; i++ {
		_, unsub, err := hub.Subscribe(sessionID, "owner")
		if err != nil {
			t.Fatalf("expected Subscribe to succeed for subscriber %d, got: %v", i+1, err)
		}
		unsubs = append(unsubs, unsub)
	}
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	// The (cap+1)th subscription must fail.
	_, _, err := hub.Subscribe(sessionID, "overflow")
	if err == nil {
		t.Fatal("expected ErrTooManySubscribers, got nil")
	}
	if err != sse.ErrTooManySubscribers {
		t.Fatalf("expected ErrTooManySubscribers, got: %v", err)
	}
}

// TestPublishDeliveredToSubscriber is a basic smoke-test: a published event
// must reach an active subscriber.
func TestPublishDeliveredToSubscriber(t *testing.T) {
	t.Parallel()

	hub := newTestHub(t)
	const sessionID = "session-delivery"

	ch, unsub, err := hub.Subscribe(sessionID, "reader")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	defer unsub()

	want := sse.Event{Type: sse.EventError, Data: sse.ErrorPayload{Message: "hello"}}
	hub.Publish(sessionID, want)

	select {
	case got := <-ch:
		if got.Type != want.Type {
			t.Fatalf("event type: got %q, want %q", got.Type, want.Type)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for event")
	}
}
