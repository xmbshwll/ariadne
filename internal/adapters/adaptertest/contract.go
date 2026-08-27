// Package adaptertest runs the shared Adapter contract against every Music
// Service adapter, so "all providers behave the same way" is one executable
// statement instead of eight hand-written approximations. Each provider test
// passes its own literal expected Capabilities: a service that silently gains or
// loses a Capability fails here with the provider named, in its own package.
package adaptertest

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/model"
)

// Contract is what one provider asserts about itself.
type Contract struct {
	// Service is the Music Service the adapter must report.
	Service model.ServiceName
	// New builds the adapter under test. Implementations must not need network
	// access or credentials: the contract only exercises declared gaps.
	New func(t *testing.T) adapters.Adapter
	// Capabilities is the literal Capability set expected, so any change to a
	// provider's declared support is a deliberate edit in that provider's test.
	Capabilities adapters.Capabilities
}

// Run asserts the shared Adapter contract: the service identity, the declared
// Capabilities, and that every method the service does not support answers with
// adapters.ErrUnsupported instead of panicking, hanging, or returning empty
// results that look like "no matches".
func Run(t *testing.T, contract Contract) {
	t.Helper()

	adapter := contract.New(t)
	require.NotNil(t, adapter)

	assert.Equal(t, contract.Service, adapter.Service())
	assert.Equal(t, contract.Capabilities, adapter.Capabilities(),
		"declared Capabilities changed; update this expectation only when the service really gained or lost the capability")

	ctx := context.Background()
	for _, gap := range unsupportedCapabilities(adapter, ctx, adapter.Capabilities()) {
		t.Run(gap.name, func(t *testing.T) {
			t.Helper()
			gap.check(t)
		})
	}
}

// unsupportedCall is one Adapter method the service does not declare, with the
// call that must answer adapters.ErrUnsupported.
type unsupportedCall struct {
	name  string
	check func(t *testing.T)
}

// unsupportedCapabilities lists the methods the service does not declare.
func unsupportedCapabilities(adapter adapters.Adapter, ctx context.Context, capabilities adapters.Capabilities) []unsupportedCall {
	assertUnsupported := func(t *testing.T, err error) {
		t.Helper()
		require.ErrorIs(t, err, adapters.ErrUnsupported,
			"unsupported methods must say so, and Target Search relies on this to skip a layer")
	}
	candidates := []struct {
		supported bool
		name      string
		check     func(t *testing.T)
	}{
		{
			supported: capabilities.AlbumUPC,
			name:      "SearchAlbumByUPC",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.SearchAlbumByUPC(ctx, "00602537184945")
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.AlbumISRC,
			name:      "SearchAlbumByISRC",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.SearchAlbumByISRC(ctx, []string{"GBAYE0601690"})
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.AlbumMetadata,
			name:      "SearchAlbumByMetadata",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.SearchAlbumByMetadata(ctx, model.CanonicalAlbum{Title: "Abbey Road", Artists: []string{"The Beatles"}})
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.SongISRC,
			name:      "SearchSongByISRC",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.SearchSongByISRC(ctx, "GBAYE0601690")
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.SongMetadata,
			name:      "SearchSongByMetadata",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.SearchSongByMetadata(ctx, model.CanonicalSong{Title: "Come Together", Artists: []string{"The Beatles"}})
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.AlbumSource,
			name:      "ParseAlbumURL",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.ParseAlbumURL("https://example.com/album/1")
				assertUnsupported(t, err)
			},
		},
		{
			supported: capabilities.SongSource,
			name:      "ParseSongURL",
			check: func(t *testing.T) {
				t.Helper()
				_, err := adapter.ParseSongURL("https://example.com/song/1")
				assertUnsupported(t, err)
			},
		},
	}

	unsupported := make([]unsupportedCall, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.supported {
			continue
		}
		unsupported = append(unsupported, unsupportedCall{name: candidate.name, check: candidate.check})
	}
	return unsupported
}
