package nexora

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

type dependencyTestLifecycle struct{}

func (dependencyTestLifecycle) Initialize(_ context.Context) error {
	return nil
}

func (dependencyTestLifecycle) Start(_ context.Context) error {
	return nil
}

func (dependencyTestLifecycle) Stop(_ context.Context) error {
	return nil
}

func (dependencyTestLifecycle) Release(_ context.Context) error {
	return nil
}

func dependencyTestRegistration(name string, dependencies ...string) Registration {
	return Registration{
		Name:      name,
		Component: dependencyTestLifecycle{},
		DependsOn: dependencies,
	}
}

func registrationNames(registrations []Registration) []string {
	names := make([]string, 0, len(registrations))

	for _, registration := range registrations {
		names = append(names, registration.Name)
	}

	return names
}

func TestOrderRegistrationsDependencyRegisteredBeforeDependent(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("database"),
		dependencyTestRegistration("repository", "database"),
		dependencyTestRegistration("api", "repository"),
	}

	ordered, err := orderRegistrations(registrations)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	expected := []string{"database", "repository", "api"}
	actual := registrationNames(ordered)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected order: got %v, want %v", actual, expected)
	}
}

func TestOrderRegistrationsDependencyRegisteredAfterDependent(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("api", "repository"),
		dependencyTestRegistration("repository", "database"),
		dependencyTestRegistration("database"),
	}

	ordered, err := orderRegistrations(registrations)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	expected := []string{"database", "repository", "api"}
	actual := registrationNames(ordered)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected order: got %v, want %v", actual, expected)
	}
}

func TestOrderRegistrationsThreeLevelGraph(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("application", "service"),
		dependencyTestRegistration("service", "repository"),
		dependencyTestRegistration("repository", "database"),
		dependencyTestRegistration("database"),
	}

	ordered, err := orderRegistrations(registrations)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	expected := []string{
		"database",
		"repository",
		"service",
		"application",
	}

	actual := registrationNames(ordered)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected order: got %v, want %v", actual, expected)
	}
}

func TestOrderRegistrationsMultipleDependencies(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("application", "cache", "database"),
		dependencyTestRegistration("cache"),
		dependencyTestRegistration("database"),
	}

	ordered, err := orderRegistrations(registrations)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	expected := []string{"cache", "database", "application"}
	actual := registrationNames(ordered)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected order: got %v, want %v", actual, expected)
	}
}

func TestOrderRegistrationsIndependtComponentsAreDeterministic(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("logger"),
		dependencyTestRegistration("metrics"),
		dependencyTestRegistration("tracing"),
		dependencyTestRegistration("diagnostics"),
	}

	for attempt := 0; attempt < 10; attempt++ {
		ordered, err := orderRegistrations(registrations)
		if err != nil {
			t.Fatalf("attempt %d: orderRegistrations() returned an error: %v", attempt, err)
		}

		expected := []string{"logger", "metrics", "tracing", "diagnostics"}
		actual := registrationNames(ordered)

		if !reflect.DeepEqual(actual, expected) {
			t.Fatalf(
				"attempt %d: unexpected order: got %v, want %v",
				attempt,
				actual,
				expected,
			)
		}
	}
}

func TestOrderRegistrationsDeterministicAmongEligibleComponents(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("api", "database"),
		dependencyTestRegistration("logger"),
		dependencyTestRegistration("database"),
		dependencyTestRegistration("metrics"),
	}

	ordered, err := orderRegistrations(registrations)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	expected := []string{"logger", "database", "api", "metrics"}
	actual := registrationNames(ordered)

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("unexpected order: got %v, want %v", actual, expected)
	}
}

func TestOrderRegistrationsRejectsCycle(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("api", "service"),
		dependencyTestRegistration("service", "repository"),
		dependencyTestRegistration("repository", "api"),
	}

	_, err := orderRegistrations(registrations)
	if err == nil {
		t.Fatal("expected dependency cycle error, got nil")
	}

	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOrderRegistrationsRejectsUnknownDependency(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("api", "missing"),
	}

	_, err := orderRegistrations(registrations)
	if err == nil {
		t.Fatal("expected unknown dependecy error, got nil")
	}
}

func TestOrderRegistrationRejectsSelfDependecy(t *testing.T) {
	registrations := []Registration{
		dependencyTestRegistration("api", "api"),
	}

	_, err := orderRegistrations(registrations)
	if err == nil {
		t.Fatal("expected self-dependency error, got nil")
	}
}

func TestOrderRegistrationsEmtyComposition(t *testing.T) {
	ordered, err := orderRegistrations(nil)
	if err != nil {
		t.Fatalf("orderRegistrations() returned an error: %v", err)
	}

	if len(ordered) != 0 {
		t.Fatalf("expected empty ordering, got %v", registrationNames(ordered))
	}
}
