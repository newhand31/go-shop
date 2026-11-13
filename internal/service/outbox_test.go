package service

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/newhand31/go-shop/internal/repository"
)

// fakeOutboxStore hands the dispatcher one canned message per Dispatch call and records what send answered.
type fakeOutboxStore struct {
	msg     repository.OutboxMessage
	calls   atomic.Int32
	sendErr error
}

func (f *fakeOutboxStore) Dispatch(ctx context.Context, _, _ int, _ func(int) time.Duration,
	send func(context.Context, repository.OutboxMessage) error) (int, error) {
	f.calls.Add(1)
	f.sendErr = send(ctx, f.msg)
	if f.sendErr != nil {
		return 0, nil
	}
	return 1, nil
}

// fakeMailer is the test double the Mailer interface exists for: it records the call and fails on demand.
type fakeMailer struct {
	orderID, total int
	err            error
}

func (f *fakeMailer) SendOrderConfirmation(_ context.Context, orderID, totalTWD int) error {
	f.orderID, f.total = orderID, totalTWD
	return f.err
}

func TestOutboxDispatcher_DecodesPayloadAndCallsMailer(t *testing.T) {
	store := &fakeOutboxStore{msg: repository.OutboxMessage{
		ID: 1, Topic: repository.TopicOrderConfirmation, Payload: []byte(`{"order_id":42,"total_twd":1290}`),
	}}
	mailer := &fakeMailer{}
	NewOutboxDispatcher(store, mailer, OutboxConfig{}, quiet()).RunOnce(context.Background())

	if mailer.orderID != 42 || mailer.total != 1290 {
		t.Fatalf("mailer got order %d total %d, want 42 / 1290", mailer.orderID, mailer.total)
	}
	if store.sendErr != nil {
		t.Fatalf("send returned %v", store.sendErr)
	}
}

func TestOutboxDispatcher_MailerFailureIsReturnedToTheStore(t *testing.T) {
	boom := errors.New("smtp down")
	store := &fakeOutboxStore{msg: repository.OutboxMessage{
		Topic: repository.TopicOrderConfirmation, Payload: []byte(`{"order_id":1,"total_twd":1}`),
	}}
	NewOutboxDispatcher(store, &fakeMailer{err: boom}, OutboxConfig{}, quiet()).RunOnce(context.Background())

	if !errors.Is(store.sendErr, boom) {
		t.Fatalf("send returned %v, want it to wrap the mailer error so the store schedules a retry", store.sendErr)
	}
}

func TestOutboxDispatcher_BadRowsAreErrorsNotPanics(t *testing.T) {
	for name, msg := range map[string]repository.OutboxMessage{
		"unknown topic": {Topic: "carrier_pigeon", Payload: []byte(`{}`)},
		"broken json":   {Topic: repository.TopicOrderConfirmation, Payload: []byte(`{`)},
	} {
		t.Run(name, func(t *testing.T) {
			store := &fakeOutboxStore{msg: msg}
			NewOutboxDispatcher(store, &fakeMailer{}, OutboxConfig{}, quiet()).RunOnce(context.Background())
			if store.sendErr == nil {
				t.Fatal("want an error so the row is retried or parked, got nil")
			}
		})
	}
}

// The lifecycle test: Run must return once its context is cancelled. If it did not, "go d.Run(ctx)" would leak a
// goroutine on every shutdown and main would hang waiting for it.
func TestOutboxDispatcher_RunStopsWhenContextIsCancelled(t *testing.T) {
	store := &fakeOutboxStore{msg: repository.OutboxMessage{Topic: "noop"}}
	d := NewOutboxDispatcher(store, &fakeMailer{}, OutboxConfig{Interval: time.Millisecond}, quiet())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		d.Run(ctx)
	}()

	for store.calls.Load() < 3 { // it really is looping
		time.Sleep(time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

func TestExponentialBackoff(t *testing.T) {
	for attempts, want := range map[int]time.Duration{
		0: time.Second, 1: time.Second, 2: 2 * time.Second, 4: 8 * time.Second, 9: 256 * time.Second, 10: 5 * time.Minute, 50: 5 * time.Minute,
	} {
		if got := exponentialBackoff(attempts); got != want {
			t.Errorf("exponentialBackoff(%d) = %v, want %v", attempts, got, want)
		}
	}
}
