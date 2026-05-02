package ariadne

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithClientBuildsDefaultServiceAdaptersOnce(t *testing.T) {
	buildCount := 0
	originalCatalog := defaultProviderCatalog
	defaultProviderCatalog = newProviderCatalog([]serviceBinding{{
		capability: serviceCapability{name: ServiceName("fixture")},
		build: func(*http.Client, Config) serviceAdapterSet {
			buildCount++
			return serviceAdapterSet{}
		},
	}}, serviceOrder{})
	t.Cleanup(func() {
		defaultProviderCatalog = originalCatalog
	})

	resolver := NewWithClient(&http.Client{}, DefaultConfig())
	require.NotNil(t, resolver)
	assert.Equal(t, 1, buildCount)
}
