package nexora

import (
	"context"
	"errors"
	"fmt"
)

func initializeLifecycle(
	ctx context.Context,
	components []Lifecycle,
) error {
	for _, component := range components {
		if err := component.Initialize(ctx); err != nil {
			return err
		}
	}

	return nil
}

func startLifecycle(
	ctx context.Context,
	components []Lifecycle,
) error {
	for _, component := range components {
		if err := component.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func stopLifecycle(
	ctx context.Context,
	components []Lifecycle,
) error {
	var errs []error

	for i := len(components) - 1; i >= 0; i-- {
		if err := components[i].Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("Stop component %d: %w", i, err))
		}
	}

	return errors.Join(errs...)
}

func releaseLifecycle(
	ctx context.Context,
	components []Lifecycle,
) error {
	var errs []error

	for i := len(components) - 1; i >= 0; i-- {
		if err := components[i].Release(ctx); err != nil {
			errs = append(errs, fmt.Errorf("release component %d: %w", i, err))
		}
	}

	return errors.Join(errs...)
}
