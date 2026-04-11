package events

import (
	"testing"
	"time"
)

func TestBusPublishDeliversExactSuffixAndAll(t *testing.T) {
	bus := NewBus(0)

	exactCh := make(chan Event, 1)
	suffixCh := make(chan Event, 1)
	allCh := make(chan Event, 1)

	bus.Subscribe(V1TodoDue, func(evt Event) { exactCh <- evt })
	bus.SubscribeSuffix(".due", func(evt Event) { suffixCh <- evt })
	bus.SubscribeAll(func(evt Event) { allCh <- evt })

	bus.Publish(Event{Name: V1TodoDue, Source: "test"})

	for label, ch := range map[string]chan Event{
		"exact":  exactCh,
		"suffix": suffixCh,
		"all":    allCh,
	} {
		select {
		case evt := <-ch:
			if evt.Name != V1TodoDue {
				t.Fatalf("%s: unexpected name %q", label, evt.Name)
			}
		case <-time.After(time.Second):
			t.Fatalf("%s: event was not delivered", label)
		}
	}
}

func TestBusRecoverEnqueuesDeadLetter(t *testing.T) {
	bus := NewBus(1)
	bus.Subscribe(V1SystemStartup, func(Event) {
		panic("boom")
	})

	bus.Publish(Event{Name: V1SystemStartup, Source: "test"})

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if bus.DeadLetterCount() == 1 {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("dead-letter was not recorded")
}
