package adapterutil_test

import (
	"errors"
	"testing"

	adapterutil "github.com/xmbshwll/ariadne/internal/adapters/adapterutil"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestRuntimeDeferredErrorMatchesSharedAndServiceSentinels(t *testing.T) {
	err := adapterutil.NewRuntimeDeferredError(model.ServiceYouTubeMusic, "song metadata fetch is deferred")

	assert.ErrorIs(t, err, adapterutil.ErrRuntimeDeferred)
	assert.ErrorIs(t, err, adapterutil.RuntimeDeferredService(model.ServiceYouTubeMusic))
	assert.False(t, errors.Is(err, adapterutil.RuntimeDeferredService(model.ServiceAmazonMusic)))
}
