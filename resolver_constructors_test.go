package ariadne

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithClientBuildsDefaultServiceAdaptersOnce(t *testing.T) {
	buildCount := 0
	originalBindings := defaultServiceBindings
	defaultServiceBindings = []serviceBinding{{
		capability: serviceCapability{name: ServiceName("fixture")},
		build: func(*http.Client, Config) serviceAdapterSet {
			buildCount++
			return serviceAdapterSet{}
		},
	}}
	t.Cleanup(func() {
		defaultServiceBindings = originalBindings
	})

	resolver := NewWithClient(&http.Client{}, DefaultConfig())
	require.NotNil(t, resolver)
	assert.Equal(t, 1, buildCount)
}
