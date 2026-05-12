package ariadne

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildResolverAdaptersBuildsServiceAdapterSetsOnce(t *testing.T) {
	buildCount := 0
	bindings := []serviceBinding{{
		capability: serviceCapability{name: ServiceName("fixture")},
		build: func(*http.Client, Config) serviceAdapterSet {
			buildCount++
			return serviceAdapterSet{}
		},
	}}

	sets := buildServiceAdapterSets(&http.Client{}, DefaultConfig(), bindings)
	require.NotNil(t, sets)
	assert.Equal(t, 1, buildCount)
}
