package adapterutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestRuntimeDeferredErrorMatchesSharedAndServiceSentinels(t *testing.T) {
	err := NewRuntimeDeferredError(model.ServiceYouTubeMusic, "song metadata fetch is deferred")

	assert.ErrorIs(t, err, ErrRuntimeDeferred)
	assert.ErrorIs(t, err, RuntimeDeferredService(model.ServiceYouTubeMusic))
	assert.False(t, errors.Is(err, RuntimeDeferredService(model.ServiceAmazonMusic)))
}
