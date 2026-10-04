package nexora

import (
	"context"
	"strings"
	"testing"
)

func TestRegisterAcceptsNamedComponentAndDependencies(t *testing.T) {
	app := New()

	if err := app.Register(Registration{
		Name:      "database",
		Component: &testLifecycle{},
	}); err != nil {
		t.Fatalf("register database: %v", err)
	}

	if err := app.Register(Registration{
		Name:      "api",
		Component: &testLifecycle{},
		DependsOn: []string{"database"},
	}); err != nil {
		t.Fatalf("register api: %v", err)
	}

	if len(app.registrations) != 2 {
		t.Fatalf(
			"expected 2 registrations, got %d",
			len(app.registrations),
		)
	}
}

func TestRegisterRejectsInvalidRegistrations(t *testing.T) {
	tests := []struct {
		name         string
		registration Registration
		want         string
	}{
		{
			name: "empty name",
			registration: Registration{
				Component: &testLifecycle{},
			},
			want: "name must not be empty",
		},
		{
			name: "nil component",
			registration: Registration{
				Name: "nil",
			},
			want: "must not be nil",
		},
		{
			name: "typed nil component",
			registration: Registration{
				Name:      "typed-nil",
				Component: (*testLifecycle)(nil),
			},
			want: "must not be nil",
		},
		{
			name: "empty dependency",
			registration: Registration{
				Name:      "api",
				Component: &testLifecycle{},
				DependsOn: []string{""},
			},

			want: "dependency name must not be empty",
		},
		{
			name: "self dependency",
			registration: Registration{
				Name:      "api",
				Component: &testLifecycle{},
				DependsOn: []string{"api"},
			},

			want: "cannot depend on itself",
		},
		{
			name: "duplicate dependency",
			registration: Registration{
				Name:      "api",
				Component: &testLifecycle{},
				DependsOn: []string{"database", "database"},
			},

			want: "declared more than once",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New().Register(tt.registration)

			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf(
					"expected error containing %q, got %v",
					tt.want,
					err,
				)
			}
		})
	}
}

func TestRegisterRejectsDuplicateNames(t *testing.T) {
	app := New()

	for i := 0; i < 2; i++ {
		err := app.Register(Registration{
			Name:      "database",
			Component: &testLifecycle{},
		})

		if i == 0 && err != nil {
			t.Fatalf("first registration failed: %v", err)
		}

		if i == 1 && (err == nil || !strings.Contains(err.Error(), "already registered")) {
			t.Fatalf("expected duplicate-name error, got %v", err)
		}
	}
}

func TestRunRejectsUnknownDependencyBeforeLifecycle(t *testing.T) {
	var events []string

	app := New()

	if err := app.Register(Registration{
		Name: "api",
		Component: &appTestComponent{
			name:   "api",
			events: &events,
		},
		DependsOn: []string{"database"},
	}); err != nil {
		t.Fatal(err)
	}

	err := app.Run(context.Background())

	if err == nil || !strings.Contains(err.Error(), "unregistered component") {
		t.Fatalf("expected unknown dependency error, got %v", err)
	}

	if len(events) != 0 {
		t.Fatalf("lifecycle ran before composition validation: %v", events)
	}
}

func TestRegisterCopiesDependencySlice(t *testing.T) {
	dependencies := []string{"database"}

	app := New()

	if err := app.Register(Registration{
		Name:      "api",
		Component: &testLifecycle{},
		DependsOn: dependencies,
	}); err != nil {
		t.Fatal(err)
	}

	dependencies[0] = "mutated"

	if got := app.registrations[0].DependsOn[0]; got != "database" {
		t.Fatalf("registration dependency changed through caller slice: %q", got)
	}
}
