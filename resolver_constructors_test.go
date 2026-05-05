package ariadne

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithProviderCatalogBuildsServiceAdaptersOnce(t *testing.T) {
	buildCount := 0
	catalog := newProviderCatalog([]serviceBinding{{
		capability: serviceCapability{name: ServiceName("fixture")},
		build: func(*http.Client, Config) serviceAdapterSet {
			buildCount++
			return serviceAdapterSet{}
		},
	}}, serviceOrder{})

	resolver := newWithProviderCatalog(&http.Client{}, DefaultConfig(), catalog)
	require.NotNil(t, resolver)
	assert.Equal(t, 1, buildCount)
}
