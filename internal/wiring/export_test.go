package wiring

import (
	"github.com/xmbshwll/ariadne/internal/model"
)

// Test-only seams for the external wiring_test package: the binding and adapter-set
// shapes are internal wiring detail with no reason to be exported.
type (
	// Binding is the built-in service binding shape under test.
	Binding = binding
	// BuiltAdapterSet is one service's built adapter set. It is not the public
	// ariadne.AdapterSet, which carries caller-supplied adapters.
	BuiltAdapterSet = adapterSet
)

var (
	// NewBinding builds a single-service binding for wiring tests.
	NewBinding = func(name model.ServiceName, build adapterBuilder) binding {
		return binding{capability: capabilitySpec{name: name}, build: build}
	}
	// BuildAdapterSets exposes the one-per-service adapter construction step.
	BuildAdapterSets = buildAdapterSets
)
