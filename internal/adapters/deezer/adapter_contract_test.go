package deezer_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/adapters/adaptertest"
	"github.com/xmbshwll/ariadne/internal/adapters/deezer"
	"github.com/xmbshwll/ariadne/internal/model"
)

// TestAdapterContract pins the shared Adapter contract for this service: the
// Capability set it declares, and adapters.ErrUnsupported from every method it
// does not declare.
func TestAdapterContract(t *testing.T) {
	adaptertest.Run(t, adaptertest.Contract{
		Service: model.ServiceDeezer,
		New: func(t *testing.T) adapters.Adapter {
			t.Helper()
			adapter := deezer.New(nil)
			require.NotNil(t, adapter)
			return adapter
		},
		Capabilities: adapters.Capabilities{
			AlbumSource:   true,
			AlbumUPC:      true,
			AlbumISRC:     true,
			AlbumMetadata: true,
			SongSource:    true,
			SongISRC:      true,
			SongMetadata:  true,
		},
	})
}
