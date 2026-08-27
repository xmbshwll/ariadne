package wiring

import (
	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

// Test-only seams for the external wiring_test package: the binding and adapter-set
// shapes are internal wiring detail with no reason to be exported.
type (
	// Binding is the built-in service binding shape under test.
	Binding = binding
	// BuiltAdapter is one service's built adapter.
	BuiltAdapter = adapters.Adapter
)

var (
	// NewBinding builds a single-service binding for wiring tests.
	NewBinding = func(name model.ServiceName, build adapterBuilder) binding {
		return binding{capability: capabilitySpec{name: name}, build: build}
	}
	// BuildAdapters exposes the one-per-service adapter construction step.
	BuildAdapters = buildAdapters
)
