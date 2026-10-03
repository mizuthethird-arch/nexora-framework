package nexora

import (
	"context"
	"testing"
)

type testLifecycle struct {
	initialized bool
	started     bool
	stopped     bool
	released    bool
}

func (t *testLifecycle) Initialize(context.Context) error {
	t.initialized = true
	return nil
}

func (t *testLifecycle) Start(context.Context) error {
	t.started = true
	return nil
}

func (t *testLifecycle) Stop(context.Context) error {
	t.stopped = true
	return nil
}

func (t *testLifecycle) Release(context.Context) error {
	t.released = true
	return nil
}

func TestLifecycle(t *testing.T) {
	component := &testLifecycle{}
	ctx := context.Background()

	if err := component.Initialize(ctx); err != nil {
		t.Fatal(err)
	}

	if err := component.Start(ctx); err != nil {
		t.Fatal(err)
	}

	if err := component.Stop(ctx); err != nil {
		t.Fatal(err)
	}

	if err := component.Release(ctx); err != nil {
		t.Fatal(err)
	}

	if !component.initialized {
		t.Fatal("component was not initialized")
	}

	if !component.started {
		t.Fatal("component was not started")
	}

	if !component.stopped {
		t.Fatal("component was not stopped")
	}

	if !component.released {
		t.Fatal("component was not released")
	}
}

func TestLifecycleOrdering(t *testing.T) {
	var events []string

	componentA := &recordingLifecycle{
		name:   "A",
		events: &events,
	}

	componentB := &recordingLifecycle{
		name:   "B",
		events: &events,
	}

	componentC := &recordingLifecycle{
		name:   "C",
		events: &events,
	}

	ctx := context.Background()

	components := []Lifecycle{
		componentA,
		componentB,
		componentC,
	}

	if err := initializeLifecycle(ctx, components); err != nil {
		t.Fatal(err)
	}

	if err := startLifecycle(ctx, components); err != nil {
		t.Fatal(err)
	}

	if err := stopLifecycle(ctx, components); err != nil {
		t.Fatal(err)
	}

	if err := releaseLifecycle(ctx, components); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"A.initialize",
		"B.initialize",
		"C.initialize",

		"A.start",
		"B.start",
		"C.start",

		"C.stop",
		"B.stop",
		"A.stop",

		"C.release",
		"B.release",
		"A.release",
	}

	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(events), events)
	}

	for i := range expected {
		if events[i] != expected[i] {
			t.Fatalf(
				"event %d: expected %q, got %q",
				i,
				expected[i],
				events[i],
			)
		}
	}
}
