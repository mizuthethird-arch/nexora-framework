package nexora

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type Registration struct {
	Name      string
	Component Lifecycle
	DependsOn []string
}

func validateRegistration(registration Registration) error {
	if strings.TrimSpace(registration.Name) == "" {
		return errors.New("name must not be empty")
	}

	if isNilLifecycle(registration.Component) {
		return fmt.Errorf(
			"register component %q: lifecycle component must not be nil",
			registration.Name,
		)
	}

	seen := make(map[string]struct{}, len(registration.DependsOn))

	for _, dependency := range registration.DependsOn {
		if strings.TrimSpace(dependency) == "" {
			return fmt.Errorf(
				"register component %q: dependency name must not be empty",
				registration.Name,
			)
		}

		if dependency == registration.Name {
			return fmt.Errorf(
				"register component %q: component cannot depend on itself",
				registration.Name,
			)
		}

		if _, exists := seen[dependency]; exists {
			return fmt.Errorf(
				"register component %q: dependency %q is declared more than once",
				registration.Name,
				dependency,
			)
		}

		seen[dependency] = struct{}{}
	}

	return nil
}

func validateComposition(registrations []Registration) error {
	names := make(map[string]struct{}, len(registrations))

	// Validate individual registrations and detect duplicates names
	for _, registration := range registrations {
		if err := validateRegistration(registration); err != nil {
			return err
		}

		if _, exists := names[registration.Name]; exists {
			return fmt.Errorf(
				"component %q is registered more than once",
				registration.Name,
			)
		}

		names[registration.Name] = struct{}{}
	}

	// Validate that every declared dependency exists
	for _, registration := range registrations {
		for _, dependency := range registration.DependsOn {
			if _, exists := names[dependency]; !exists {
				return fmt.Errorf(
					"component %q depends on unregistered component %q",
					registration.Name,
					dependency,
				)
			}
		}
	}

	return nil
}

func isNilLifecycle(component Lifecycle) bool {
	if component == nil {
		return true
	}

	value := reflect.ValueOf(component)

	switch value.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:

		return value.IsNil()

	default:
		return false
	}
}
