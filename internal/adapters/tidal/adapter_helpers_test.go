package tidal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncludedResourceLookupsUseTypeAndID(t *testing.T) {
	included := []apiResource{
		{ID: "shared", Type: "albums", Attributes: resourceAttributes{Title: "Album Resource"}},
		{ID: "shared", Type: "artists", Attributes: resourceAttributes{Name: "Artist Resource"}},
		{ID: "shared", Type: "artworks", Attributes: resourceAttributes{Files: []resourceFile{{Href: "https://resources.tidal.test/shared.jpg", Meta: fileMeta{Width: 1280, Height: 1280}}}}},
	}

	resourceByID := includedResourceIndex(included)

	artistNames := includedArtistNames(resourceByID, []relationshipData{{ID: "shared", Type: "artists"}})
	assert.Equal(t, []string{"Artist Resource"}, artistNames)

	album := firstRelatedResource(resourceByID, []relationshipData{{ID: "shared", Type: "albums"}}, "albums")
	require.NotNil(t, album)
	assert.Equal(t, "Album Resource", album.Attributes.Title)

	artworkURL := artworkURLFromIncluded(resourceByID, []relationshipData{{ID: "shared", Type: "artworks"}})
	assert.Equal(t, "https://resources.tidal.test/shared.jpg", artworkURL)
}

func TestAlbumIDsFromTrackDocumentMergesIncludedAndRelationshipIDs(t *testing.T) {
	document := apiDocument{
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
		Included: []apiResource{{ID: "included-album", Type: "albums"}},
	}

	assert.Equal(t, []string{"included-album", "relationship-album"}, albumIDsFromTrackDocument(document))
}
