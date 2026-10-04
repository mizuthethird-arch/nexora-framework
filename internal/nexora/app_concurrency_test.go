package nexora

import (
	"context"
	"strings"
	"testing"
)

type signalingLifecycle struct {
	initialized chan struct{}
}

func (s *signalingLifecycle) Initialize(context.Context) error {
	close(s.initialized)
	return nil
}

func (s *signalingLifecycle) Start(context.Context) error {
	return nil
}

func (s *signalingLifecycle) Stop(context.Context) error {
	return nil
}

func (s *signalingLifecycle) Release(context.Context) error {
	return nil
}

func TestRegisterRejectedAfterRunStarts(t *testing.T) {
	app := New()

	initialized := make(chan struct{})

	if err := app.Register(Registration{
		Name: "database",
		Component: &signalingLifecycle{
			initialized: initialized,
		},
	}); err != nil {
		t.Fatalf("register database: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runResult := make(chan error, 1)

	go func() {
		runResult <- app.Run(ctx)
	}()

	<-initialized

	err := app.Register(Registration{
		Name:      "api",
		Component: &testLifecycle{},
	})

	if err == nil {
		t.Fatal("expected registration to be rejected after Run started")
	}

	if !strings.Contains(err.Error(), "registration is closed") {
		t.Fatalf("unexpected registration error: %v", err)
	}

	cancel()

	if err := <-runResult; err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestAddIgnoredAfterRunStarts(t *testing.T) {
	app := New()

	initialized := make(chan struct{})

	if err := app.Register(Registration{
		Name: "database",
		Component: &signalingLifecycle{
			initialized: initialized,
		},
	}); err != nil {
		t.Fatalf("register database: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runResult := make(chan error, 1)

	go func() {
		runResult <- app.Run(ctx)
	}()

	<-initialized

	app.mu.Lock()
	registrationCount := len(app.registrations)
	app.mu.Unlock()

	if registrationCount != 1 {
		t.Fatalf(
			"expected Add to be ignored, got %d registrations",
			registrationCount,
		)
	}

	cancel()

	if err := <-runResult; err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestRunCannotBeStartedTwice(t *testing.T) {
	app := New()

	initialized := make(chan struct{})

	if err := app.Register(Registration{
		Name: "database",
		Component: &signalingLifecycle{
			initialized: initialized,
		},
	}); err != nil {
		t.Fatalf("register database: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runResult := make(chan error, 1)

	go func() {
		runResult <- app.Run(ctx)
	}()

	<-initialized

	err := app.Run(context.Background())

	if err == nil {
		t.Fatal("expected second Run call to be rejected")
	}

	if !strings.Contains(err.Error(), "already been started") {
		t.Fatalf("unexpected second Run error: %v", err)
	}

	cancel()

	if err := <-runResult; err == nil {
		t.Fatal("expected context cancellation error")
	}
}

func TestRegistrationAllowedAfterCompositionValidationFailure(t *testing.T) {
	app := New()

	if err := app.Register(Registration{
		Name:      "api",
		Component: &testLifecycle{},
		DependsOn: []string{"database"},
	}); err != nil {
		t.Fatal(err)
	}

	err := app.Run(context.Background())

	if err == nil {
		t.Fatal("expected composition validation failure")
	}

	// A validation failure occurs vefore Run starts
	// The application must remain configurable
	err = app.Register(Registration{
		Name:      "database",
		Component: &testLifecycle{},
	})

	if err != nil {
		t.Fatalf(
			"expected registration to remain open after validation failure: %v",
			err,
		)
	}
}
