package ariadne

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errUnsupportedLibrarySource = errors.New("unsupported")

func TestLoadConfigFromEnv(t *testing.T) {
	config := LoadConfigFromEnv(func(key string) string {
		switch key {
		case "SPOTIFY_CLIENT_ID":
			return " spotify-client "
		case "SPOTIFY_CLIENT_SECRET":
			return " spotify-secret "
		case "APPLE_MUSIC_STOREFRONT":
			return " GB "
		case "APPLE_MUSIC_KEY_ID":
			return " music-key "
		case "APPLE_MUSIC_TEAM_ID":
			return " team-id "
		case "APPLE_MUSIC_PRIVATE_KEY_PATH":
			return " /tmp/AuthKey_TEST.p8 "
		case "TIDAL_CLIENT_ID":
			return " tidal-client "
		case "TIDAL_CLIENT_SECRET":
			return " tidal-secret "
		case "ARIADNE_HTTP_TIMEOUT":
			return " 45s "
		default:
			return ""
		}
	})

	assert.Equal(t, "gb", config.AppleMusicStorefront)
	assert.Equal(t, "spotify-client", config.Spotify.ClientID)
	assert.Equal(t, "spotify-secret", config.Spotify.ClientSecret)
	assert.Equal(t, "music-key", config.AppleMusic.KeyID)
	assert.Equal(t, "team-id", config.AppleMusic.TeamID)
	assert.Equal(t, "/tmp/AuthKey_TEST.p8", config.AppleMusic.PrivateKeyPath)
	assert.Equal(t, "tidal-client", config.TIDAL.ClientID)
	assert.Equal(t, "tidal-secret", config.TIDAL.ClientSecret)
	assert.Equal(t, 45*time.Second, config.HTTPTimeout)
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.Equal(t, "us", config.AppleMusicStorefront)
	assert.NotEqual(t, ScoreWeights{}, config.ScoreWeights)
	assert.NotEqual(t, SongScoreWeights{}, config.SongScoreWeights)
	assert.Equal(t, 15*time.Second, config.HTTPTimeout)
}

func TestNormalizedConfigDefaultsSongWeights(t *testing.T) {
	config := normalizedConfig(Config{})
	assert.NotEqual(t, SongScoreWeights{}, config.SongScoreWeights)
}

func TestMatchStrengthForScore(t *testing.T) {
	tests := []struct {
		score int
		want  MatchStrength
	}{
		{score: 120, want: MatchStrengthStrong},
		{score: 80, want: MatchStrengthProbable},
		{score: 50, want: MatchStrengthWeak},
		{score: 49, want: MatchStrengthVeryWeak},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.want, MatchStrengthForScore(tt.score))
	}
}

func TestNewWithAdaptersResolveAlbum(t *testing.T) {
	resolver := NewWithAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://fixture.test/source")
	require.NoError(t, err)
	assert.Equal(t, ServiceDeezer, resolution.Source.Service)
	match := resolution.Matches[ServiceSpotify]
	require.NotNil(t, match.Best)
	assert.Equal(t, "spotify-1", match.Best.Candidate.CandidateID)
}

func TestNewWithEntityAdaptersResolveSong(t *testing.T) {
	resolver := NewWithEntityAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
		[]SongSourceAdapter{librarySongSourceAdapter{}},
		[]SongTargetAdapter{librarySongTargetAdapter{}},
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	require.NoError(t, err)
	assert.Equal(t, ServiceSpotify, resolution.Source.Service)
	match := resolution.Matches[ServiceAppleMusic]
	require.NotNil(t, match.Best)
	assert.Equal(t, "apple-song-1", match.Best.Candidate.CandidateID)
}

func TestResolverResolveDispatchesByEntityType(t *testing.T) {
	resolver := NewWithEntityAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
		[]SongSourceAdapter{librarySongSourceAdapter{}},
		[]SongTargetAdapter{librarySongTargetAdapter{}},
	)

	albumEntity, err := resolver.Resolve(context.Background(), "https://fixture.test/source")
	require.NoError(t, err)
	require.NotNil(t, albumEntity.Album)
	assert.Nil(t, albumEntity.Song)
	assert.Equal(t, "album", albumEntity.Parsed.EntityType)

	songEntity, err := resolver.Resolve(context.Background(), "https://fixture.test/songs/1")
	require.NoError(t, err)
	require.NotNil(t, songEntity.Song)
	assert.Nil(t, songEntity.Album)
	assert.Equal(t, "song", songEntity.Parsed.EntityType)
}

type librarySourceAdapter struct{}

func (librarySourceAdapter) Service() ServiceName {
	return ServiceDeezer
}

func (librarySourceAdapter) ParseAlbumURL(raw string) (*ParsedAlbumURL, error) {
	if raw != "https://fixture.test/source" {
		return nil, errUnsupportedLibrarySource
	}
	return &ParsedAlbumURL{
		Service:      ServiceDeezer,
		EntityType:   "album",
		ID:           "src-1",
		CanonicalURL: raw,
		RawURL:       raw,
	}, nil
}

func (librarySourceAdapter) FetchAlbum(_ context.Context, parsed ParsedAlbumURL) (*CanonicalAlbum, error) {
	return &CanonicalAlbum{
		Service:           parsed.Service,
		SourceID:          parsed.ID,
		SourceURL:         parsed.CanonicalURL,
		Title:             "Fixture Album",
		NormalizedTitle:   "fixture album",
		Artists:           []string{"Fixture Artist"},
		NormalizedArtists: []string{"fixture artist"},
		UPC:               "123456789012",
		TrackCount:        2,
		Tracks:            []CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
	}, nil
}

type libraryTargetAdapter struct{}

func (libraryTargetAdapter) Service() ServiceName {
	return ServiceSpotify
}

func (libraryTargetAdapter) SearchByUPC(_ context.Context, upc string) ([]CandidateAlbum, error) {
	if upc == "" {
		return nil, nil
	}
	return []CandidateAlbum{{
		CanonicalAlbum: CanonicalAlbum{
			Service:           ServiceSpotify,
			SourceID:          "spotify-1",
			SourceURL:         "https://open.spotify.com/album/spotify-1",
			Title:             "Fixture Album",
			NormalizedTitle:   "fixture album",
			Artists:           []string{"Fixture Artist"},
			NormalizedArtists: []string{"fixture artist"},
			UPC:               upc,
			TrackCount:        2,
			Tracks:            []CanonicalTrack{{Title: "Alpha", NormalizedTitle: "alpha", ISRC: "ISRC001"}, {Title: "Beta", NormalizedTitle: "beta"}},
		},
		CandidateID: "spotify-1",
		MatchURL:    "https://open.spotify.com/album/spotify-1",
	}}, nil
}

func (libraryTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]CandidateAlbum, error) {
	return nil, nil
}

func (libraryTargetAdapter) SearchByMetadata(_ context.Context, _ CanonicalAlbum) ([]CandidateAlbum, error) {
	return nil, nil
}

type librarySongSourceAdapter struct{}

func (librarySongSourceAdapter) Service() ServiceName {
	return ServiceSpotify
}

func (librarySongSourceAdapter) ParseSongURL(raw string) (*ParsedURL, error) {
	if raw != "https://fixture.test/songs/1" {
		return nil, errUnsupportedLibrarySource
	}
	return &ParsedURL{
		Service:      ServiceSpotify,
		EntityType:   "song",
		ID:           "song-1",
		CanonicalURL: raw,
		RawURL:       raw,
	}, nil
}

func (librarySongSourceAdapter) FetchSong(_ context.Context, parsed ParsedURL) (*CanonicalSong, error) {
	return &CanonicalSong{
		Service:              parsed.Service,
		SourceID:             parsed.ID,
		SourceURL:            parsed.CanonicalURL,
		Title:                "Fixture Song",
		NormalizedTitle:      "fixture song",
		Artists:              []string{"Fixture Artist"},
		NormalizedArtists:    []string{"fixture artist"},
		DurationMS:           180000,
		ISRC:                 "ISRCSONG001",
		TrackNumber:          1,
		AlbumTitle:           "Fixture Album",
		AlbumNormalizedTitle: "fixture album",
	}, nil
}

type librarySongTargetAdapter struct{}

func (librarySongTargetAdapter) Service() ServiceName {
	return ServiceAppleMusic
}

func (librarySongTargetAdapter) SearchSongByISRC(_ context.Context, isrc string) ([]CandidateSong, error) {
	if isrc == "" {
		return nil, nil
	}
	return []CandidateSong{{
		CanonicalSong: CanonicalSong{
			Service:              ServiceAppleMusic,
			SourceID:             "apple-song-1",
			SourceURL:            "https://music.apple.com/us/song/apple-song-1",
			Title:                "Fixture Song",
			NormalizedTitle:      "fixture song",
			Artists:              []string{"Fixture Artist"},
			NormalizedArtists:    []string{"fixture artist"},
			DurationMS:           180100,
			ISRC:                 isrc,
			TrackNumber:          1,
			AlbumTitle:           "Fixture Album",
			AlbumNormalizedTitle: "fixture album",
		},
		CandidateID: "apple-song-1",
		MatchURL:    "https://music.apple.com/us/song/apple-song-1",
	}}, nil
}

func (librarySongTargetAdapter) SearchSongByMetadata(_ context.Context, _ CanonicalSong) ([]CandidateSong, error) {
	return nil, nil
}
