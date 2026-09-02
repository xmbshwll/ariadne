package adapters_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errNotDeferred = errors.New("other")

func TestRuntimeDeferredErrorMatchesSharedAndServiceSentinels(t *testing.T) {
	err := adapters.NewRuntimeDeferredError(model.ServiceYouTubeMusic, "song metadata fetch is deferred")

	assert.ErrorIs(t, err, adapters.ErrRuntimeDeferred)
	assert.ErrorIs(t, err, adapters.RuntimeDeferredService(model.ServiceYouTubeMusic))
	assert.False(t, errors.Is(err, adapters.RuntimeDeferredService(model.ServiceAmazonMusic)))
}

// TestRuntimeDeferredErrorMessages pins the sentences the CLI prints when a
// recognized URL cannot be hydrated yet.
func TestRuntimeDeferredErrorMessages(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		message string
	}{
		{
			name:    "shared sentinel only",
			err:     adapters.RuntimeDeferredError{},
			message: "runtime adapter is deferred",
		},
		{
			name:    "deferred service",
			err:     adapters.RuntimeDeferredService(model.ServiceAmazonMusic),
			message: "amazon music runtime adapter is deferred",
		},
		{
			name:    "deferred service with reason",
			err:     adapters.NewRuntimeDeferredError(model.ServiceYouTubeMusic, "song metadata fetch is deferred"),
			message: "youtube music runtime adapter is deferred: song metadata fetch is deferred",
		},
		{
			name:    "service without a friendly label",
			err:     adapters.RuntimeDeferredService(model.ServiceDeezer),
			message: "deezer runtime adapter is deferred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.message, tt.err.Error())
		})
	}
}

// TestRuntimeDeferredErrorMatchesPointerService keeps the identity rule working
// for callers that compare against a pointer, and keeps an empty Service target
// from matching every deferred service.
func TestRuntimeDeferredErrorMatchesPointerService(t *testing.T) {
	err := adapters.RuntimeDeferredService(model.ServiceAmazonMusic)

	assert.ErrorIs(t, err, &adapters.RuntimeDeferredError{Service: model.ServiceAmazonMusic})
	assert.False(t, errors.Is(err, &adapters.RuntimeDeferredError{}))
	assert.False(t, errors.Is(err, errNotDeferred))
}
