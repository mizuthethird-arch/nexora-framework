package nexora

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type appState uint8

const (
	appStateConfiguring appState = iota
	appStateRunning
	appStateStopped
)

type App struct {
	mu            sync.Mutex
	state         appState
	registrations []Registration
}

func New() *App {
	return &App{
		state: appStateConfiguring,
	}
}

// Add registers a lifecycle component using an automatically generated name
func (a *App) Add(component Lifecycle) {
	if a == nil {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != appStateConfiguring {
		return
	}

	// Find an unused generated name
	var name string
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("component-%d", i)

		exists := false
		for _, registration := range a.registrations {
			if registration.Name == candidate {
				exists = true
				break
			}
		}

		if !exists {
			name = candidate
			break
		}
	}

	a.registrations = append(
		a.registrations,
		Registration{
			Name:      name,
			Component: component,
		},
	)

}

func (a *App) Register(registration Registration) error {
	if a == nil {
		return errors.New("register component: app is nil")
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.state != appStateConfiguring {
		return errors.New(
			"register component: application registration is closed",
		)
	}

	if err := validateRegistration(registration); err != nil {
		return err
	}

	for _, existing := range a.registrations {
		if existing.Name == registration.Name {
			return fmt.Errorf(
				"register component %q: name is already registered",
				registration.Name,
			)
		}
	}

	registration.DependsOn = append(
		[]string(nil),
		registration.DependsOn...,
	)

	a.registrations = append(
		a.registrations,
		registration,
	)

	return nil

}

// Run initializes, starts, supervises, and shuts down the application
func (a *App) Run(ctx context.Context) error {
	if a == nil {
		return errors.New("run application: app is nil")
	}

	if ctx == nil {
		return errors.New("run application: context must not be nil")
	}

	// Validate snapshots and trasition state atomically
	a.mu.Lock()

	if a.state != appStateConfiguring {
		a.mu.Unlock()

		return errors.New(
			"run application: application has already been started",
		)
	}

	if err := validateComposition(a.registrations); err != nil {
		a.mu.Unlock()

		return fmt.Errorf(
			"validate application composition: %w",
			err,
		)
	}

	registrations := append(
		[]Registration(nil),
		a.registrations...,
	)

	a.state = appStateRunning
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.state = appStateStopped
		a.mu.Unlock()
	}()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	components := make([]Lifecycle, 0, len(registrations))

	for _, registration := range registrations {
		components = append(
			components,
			registration.Component,
		)
	}

	initialized := make([]Lifecycle, 0, len(components))
	started := make([]Lifecycle, 0, len(components))

	// Initialization phase
	for i, component := range components {
		if err := initializeLifecycle(
			runCtx,
			[]Lifecycle{component},
		); err != nil {
			shutdownCtx := context.WithoutCancel(runCtx)

			return errors.Join(
				fmt.Errorf(
					"initialize component %q: %w",
					registrations[i].Name,
					err,
				),
				releaseLifecycle(
					shutdownCtx,
					initialized,
				),
			)
		}

		initialized = append(initialized, component)
	}

	// Startup phase
	for i, component := range initialized {
		if err := startLifecycle(
			runCtx,
			[]Lifecycle{component},
		); err != nil {
			shutdownCtx := context.WithoutCancel(runCtx)

			return errors.Join(
				fmt.Errorf(
					"start component %q: %w",
					registrations[i].Name,
					err,
				),
				stopLifecycle(
					shutdownCtx,
					started,
				),
				releaseLifecycle(
					shutdownCtx,
					initialized,
				),
			)
		}

		started = append(started, component)
	}

	<-runCtx.Done()

	shutdownCtx := context.WithoutCancel(runCtx)

	return errors.Join(
		runCtx.Err(),
		stopLifecycle(shutdownCtx, started),
		releaseLifecycle(shutdownCtx, initialized),
	)
}
