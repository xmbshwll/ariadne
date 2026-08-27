package wiring_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

func TestBuildResolverAdaptersBuildsServiceAdapterSetsOnce(t *testing.T) {
	buildCount := 0
	bindings := []wiring.Binding{wiring.NewBinding(model.ServiceName("fixture"), func(*http.Client, config.Config) wiring.BuiltAdapterSet {
		buildCount++
		return wiring.BuiltAdapterSet{}
	})}

	sets := wiring.BuildAdapterSets(&http.Client{}, config.Config{}, bindings)
	require.NotNil(t, sets)
	assert.Equal(t, 1, buildCount)
}
