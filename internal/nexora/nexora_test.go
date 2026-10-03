package nexora

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAppRunStopsWhenContextIsCacelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	app := New()

	done := make(chan error, 1)

	go func() {
		done <- app.Run(ctx)
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("application did not stop after context cancellation")
	}
}

type appTestComponent struct {
	name          string
	events        *[]string
	initializeErr error
	startErr      error
	stopErr       error
	releaseErr    error
	onStart       func()
}

func (c *appTestComponent) Initialize(ctx context.Context) error {
	*c.events = append(*c.events, c.name+".initialize")
	return c.initializeErr
}

func (c *appTestComponent) Start(ctx context.Context) error {
	*c.events = append(*c.events, c.name+".start")

	if c.onStart != nil {
		c.onStart()
	}

	return c.startErr
}

func (c *appTestComponent) Stop(ctx context.Context) error {
	*c.events = append(*c.events, c.name+".stop")
	return c.stopErr
}

func (c *appTestComponent) Release(ctx context.Context) error {
	*c.events = append(*c.events, c.name+".release")
	return c.releaseErr
}

func TestAppRunLifecycleOrdering(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var events []string

	app := New()

	app.Add(&appTestComponent{
		name:   "A",
		events: &events,
	})

	app.Add(&appTestComponent{
		name:   "B",
		events: &events,
	})

	app.Add(&appTestComponent{
		name:    "C",
		events:  &events,
		onStart: cancel,
	})

	err := app.Run(ctx)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
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
		t.Fatalf("expected %d events, got %d: %v",
			len(expected), len(events), events)
	}

	for i := range expected {
		if events[i] != expected[i] {
			t.Errorf("event %d: expected %q, got %q",
				i, expected[i], events[i])
		}
	}
}

func TestAppRunInitializationFailure(t *testing.T) {
	initErr := errors.New("initialization failed")

	var events []string

	app := New()

	app.Add(&appTestComponent{
		name:   "A",
		events: &events,
	})

	app.Add(&appTestComponent{
		name:          "B",
		events:        &events,
		initializeErr: initErr,
	})

	err := app.Run(context.Background())

	if !errors.Is(err, initErr) {
		t.Fatalf("expected initialization error, got %v", err)
	}

	expected := []string{
		"A.initialize",
		"B.initialize",
		"A.release",
	}

	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(events), events)
	}

	for i := range expected {
		if events[i] != expected[i] {
			t.Errorf("event %d: expected %q, got %q", i, expected[i], events[i])
		}
	}
}

func TestAppRunStartupFailure(t *testing.T) {
	startErr := errors.New("startup failed")

	var events []string

	app := New()

	app.Add(&appTestComponent{
		name:   "A",
		events: &events,
	})

	app.Add(&appTestComponent{
		name:     "B",
		events:   &events,
		startErr: startErr,
	})

	app.Add(&appTestComponent{
		name:   "C",
		events: &events,
	})

	err := app.Run(context.Background())

	if !errors.Is(err, startErr) {
		t.Fatalf("expected startup error, got %v", err)
	}

	expected := []string{
		"A.initialize",
		"B.initialize",
		"C.initialize",
		"A.start",
		"B.start",
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
			t.Errorf("event %d: expected %q, got %q", i, expected[i], events[i])
		}
	}
}
