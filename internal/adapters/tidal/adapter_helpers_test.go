package tidal_test

import (
	"testing"

	tidal "github.com/xmbshwll/ariadne/internal/adapters/tidal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncludedResourceLookupsUseTypeAndID(t *testing.T) {
	included := []tidal.APIResource{
		{ID: "shared", Type: "albums", Attributes: tidal.ResourceAttributes{Title: "Album Resource"}},
		{ID: "shared", Type: "artists", Attributes: tidal.ResourceAttributes{Name: "Artist Resource"}},
		{ID: "shared", Type: "artworks", Attributes: tidal.ResourceAttributes{Files: []tidal.ResourceFile{{Href: "https://resources.tidal.test/shared.jpg", Meta: tidal.FileMeta{Width: 1280, Height: 1280}}}}},
	}

	resourceByID := tidal.IncludedResourceIndex(included)

	artistNames := tidal.IncludedArtistNames(resourceByID, []tidal.RelationshipData{{ID: "shared", Type: "artists"}})
	assert.Equal(t, []string{"Artist Resource"}, artistNames)

	album := tidal.FirstRelatedResource(resourceByID, []tidal.RelationshipData{{ID: "shared", Type: "albums"}}, "albums")
	require.NotNil(t, album)
	assert.Equal(t, "Album Resource", album.Attributes.Title)

	artworkURL := tidal.ArtworkURLFromIncluded(resourceByID, []tidal.RelationshipData{{ID: "shared", Type: "artworks"}})
	assert.Equal(t, "https://resources.tidal.test/shared.jpg", artworkURL)
}

func TestAlbumIDsFromTrackDocumentMergesIncludedAndRelationshipIDs(t *testing.T) {
	document := tidal.APIDocument{
		Data: []any{map[string]any{
			"id":   "track-1",
			"type": "tracks",
			"relationships": map[string]any{
				"albums": map[string]any{
					"data": []map[string]any{
						{"id": "included-album", "type": "albums"},
						{"id": " relationship-album ", "type": "albums"},
						{"id": "wrong-type", "type": "artists"},
					},
				},
			},
		}},
		Included: []tidal.APIResource{{ID: "included-album", Type: "albums"}},
	}

	albumIDs, err := tidal.AlbumIDsFromTrackDocument(document)
	require.NoError(t, err)
	assert.Equal(t, []string{"included-album", "relationship-album"}, albumIDs)
}

func TestDocumentDataReturnsMalformedResponseErrorForUnexpectedType(t *testing.T) {
	_, err := tidal.DocumentData(tidal.APIDocument{Data: "bad"})
	require.ErrorIs(t, err, tidal.ErrMalformedTIDALAPIResponse)
}
