package nexora

import (
	"context"
	"errors"
	"fmt"
)

type App struct {
	lifecycle []Lifecycle
}

func New() *App {
	return &App{}
}

func (a *App) Add(component Lifecycle) {
	a.lifecycle = append(a.lifecycle, component)
}

func (a *App) Run(ctx context.Context) error {
	initialized := make([]Lifecycle, 0, len(a.lifecycle))
	started := make([]Lifecycle, 0, len(a.lifecycle))

	for i, component := range a.lifecycle {
		if err := initializeLifecycle(ctx, []Lifecycle{component}); err != nil {
			shutdownCtx := context.WithoutCancel(ctx)

			return errors.Join(
				fmt.Errorf("initialize component %d: %w", i, err),
				releaseLifecycle(shutdownCtx, initialized),
			)
		}

		initialized = append(initialized, component)
	}

	for i, component := range initialized {
		if err := startLifecycle(ctx, []Lifecycle{component}); err != nil {
			shutdownCtx := context.WithoutCancel(ctx)

			return errors.Join(
				fmt.Errorf("start component %d, %w", i, err),
				stopLifecycle(shutdownCtx, started),
				releaseLifecycle(shutdownCtx, initialized),
			)
		}

		started = append(started, component)
	}

	<-ctx.Done()

	shutdownCtx := context.WithoutCancel(ctx)

	return errors.Join(
		ctx.Err(),
		stopLifecycle(shutdownCtx, started),
		releaseLifecycle(shutdownCtx, initialized),
	)
}
