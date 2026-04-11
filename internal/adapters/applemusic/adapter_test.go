package applemusic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFetchAlbum(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	album, err := fixture.adapter.FetchAlbum(context.Background(), fixture.parsed)
	require.NoError(t, err)
	assert.Equal(t, abbeyRoadRemastered, album.Title)
	assert.Equal(t, "1441164426", album.SourceID)
	assert.Equal(t, "https://music.apple.com/us/album/abbey-road-remastered/1441164426", album.SourceURL)
	assert.Equal(t, 17, album.TrackCount)
	require.Len(t, album.Tracks, 17)
	assert.Equal(t, comeTogetherTitle, album.Tracks[0].Title)
	assert.Equal(t, 258947, album.Tracks[0].DurationMS)
	assert.NotEmpty(t, album.ArtworkURL)
	assert.Equal(t, "1969-09-26", album.ReleaseDate)
}

func TestFetchSong(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	song, err := fixture.adapter.FetchSong(context.Background(), model.ParsedURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   entitySong,
		ID:           "1441164430",
		CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
		RegionHint:   "us",
	})
	require.NoError(t, err)
	assert.Equal(t, comeTogetherTitle, song.Title)
	assert.Equal(t, abbeyRoadRemastered, song.AlbumTitle)
	assert.Equal(t, 1, song.TrackNumber)
	assert.Equal(t, comeTogetherISRC, song.ISRC)
}

func TestFetchSongRejectsNonSongLookupPayload(t *testing.T) {
	fixture := newTestFixture(t, buildTestPayloads(t))

	song, err := fixture.adapter.FetchSong(context.Background(), model.ParsedURL{
		Service:      model.ServiceAppleMusic,
		EntityType:   entitySong,
		ID:           "123456789",
		CanonicalURL: "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=123456789",
		RegionHint:   "us",
	})
	require.Error(t, err)
	assert.Nil(t, song)
	assert.ErrorIs(t, err, errAppleMusicSongNotFound)
}
