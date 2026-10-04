package nexora

import (
	"context"
	"errors"
	"testing"
	"time"
)

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

type cancellationOwnershipComponent struct {
	initializeCtx context.Context
	startCtx      context.Context
	startErr      error
}

func (c *cancellationOwnershipComponent) Initialize(ctx context.Context) error {
	c.initializeCtx = ctx
	return nil
}

func (c *cancellationOwnershipComponent) Start(ctx context.Context) error {
	c.startCtx = ctx
	return c.startErr
}

func (c *cancellationOwnershipComponent) Stop(ctx context.Context) error {
	return nil
}

func (c *cancellationOwnershipComponent) Release(ctx context.Context) error {
	return nil
}

func TestAppRunOwnsLifecycleCancellationContext(t *testing.T) {
	startErr := errors.New("startup failed")

	component := &cancellationOwnershipComponent{
		startErr: startErr,
	}

	app := New()
	app.Add(component)

	parentCtx := context.Background()

	err := app.Run(parentCtx)

	if !errors.Is(err, startErr) {
		t.Fatalf(
			"expected startup error, got %v",
			err,
		)
	}

	if component.initializeCtx == nil {
		t.Fatal("initialize did not receive a context")
	}

	if component.startCtx == nil {
		t.Fatal("start did not receive a context")
	}

	if parentCtx.Done() != nil {
		t.Fatal("expected background parent context to have no Done channel")
	}

	if component.initializeCtx.Done() == nil {
		t.Fatal("expected Runtime-owned context to have a Done channel")
	}

	if component.initializeCtx.Done() != component.startCtx.Done() {
		t.Fatal("initialize and start did not receive the same lifecycle context")
	}

	if !errors.Is(
		component.initializeCtx.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"expected Runtime to cancel its lifecycle context after Run returns, got %v",
			component.initializeCtx.Err(),
		)
	}
}

func TestAppRunPropagatesParentCancellation(t *testing.T) {
	parentCtx, cancel := context.WithCancel(context.Background())

	component := &cancellationOwnershipComponent{}

	app := New()
	app.Add(component)

	done := make(chan error, 1)

	go func() {
		done <- app.Run(parentCtx)
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf(
				"expected context.Canceled, got %v",
				err,
			)
		}

	case <-time.After(time.Second):
		t.Fatal("application did not stop after parent cancellation")
	}

	if component.initializeCtx == nil {
		t.Fatal("initialize did not receive a context")
	}

	if !errors.Is(
		component.initializeCtx.Err(),
		context.Canceled,
	) {
		t.Fatalf(
			"expected Runtime-owned context to be canceled, got %v",
			component.initializeCtx.Err(),
		)
	}
}

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
