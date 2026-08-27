package base_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/model"
)

// TestUnsupportedAnswersEveryMethodWithErrUnsupported pins the default a
// provider inherits when it embeds base.Unsupported: the call is refused, and
// the message names both the service and the capability it lacks.
func TestUnsupportedAnswersEveryMethodWithErrUnsupported(t *testing.T) {
	unsupported := base.Unsupported{ServiceName: model.ServiceBandcamp}
	calls := []struct {
		name       string
		capability string
		call       func() error
	}{
		{"ParseAlbumURL", "album source", func() error {
			_, err := unsupported.ParseAlbumURL("https://bandcamp.com/album/1")
			return err
		}},
		{"FetchAlbum", "album source", func() error {
			_, err := unsupported.FetchAlbum(context.Background(), model.ParsedAlbumURL{Service: model.ServiceBandcamp})
			return err
		}},
		{"ParseSongURL", "song source", func() error {
			_, err := unsupported.ParseSongURL("https://bandcamp.com/track/1")
			return err
		}},
		{"FetchSong", "song source", func() error {
			_, err := unsupported.FetchSong(context.Background(), model.ParsedURL{Service: model.ServiceBandcamp})
			return err
		}},
		{"SearchAlbumByUPC", "album UPC search", func() error {
			_, err := unsupported.SearchAlbumByUPC(context.Background(), "00602537184945")
			return err
		}},
		{"SearchAlbumByISRC", "album ISRC search", func() error {
			_, err := unsupported.SearchAlbumByISRC(context.Background(), []string{"GBAYE0601690"})
			return err
		}},
		{"SearchAlbumByMetadata", "album metadata search", func() error {
			_, err := unsupported.SearchAlbumByMetadata(context.Background(), model.CanonicalAlbum{Title: "Abbey Road"})
			return err
		}},
		{"SearchSongByISRC", "song ISRC search", func() error {
			_, err := unsupported.SearchSongByISRC(context.Background(), "GBAYE0601690")
			return err
		}},
		{"SearchSongByMetadata", "song metadata search", func() error {
			_, err := unsupported.SearchSongByMetadata(context.Background(), model.CanonicalSong{Title: "Come Together"})
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			err := call.call()
			require.ErrorIs(t, err, adapters.ErrUnsupported)
			assert.Contains(t, err.Error(), string(model.ServiceBandcamp))
			assert.Contains(t, err.Error(), call.capability)
		})
	}
}
