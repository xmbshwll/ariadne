package ariadne

import (
	"context"
	"errors"
	"testing"
	"time"
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

	if config.AppleMusicStorefront != "gb" {
		t.Fatalf("apple music storefront = %q, want gb", config.AppleMusicStorefront)
	}
	if config.Spotify.ClientID != "spotify-client" || config.Spotify.ClientSecret != "spotify-secret" {
		t.Fatalf("unexpected spotify config: %#v", config.Spotify)
	}
	if config.AppleMusic.KeyID != "music-key" || config.AppleMusic.TeamID != "team-id" || config.AppleMusic.PrivateKeyPath != "/tmp/AuthKey_TEST.p8" {
		t.Fatalf("unexpected apple music config: %#v", config.AppleMusic)
	}
	if config.TIDAL.ClientID != "tidal-client" || config.TIDAL.ClientSecret != "tidal-secret" {
		t.Fatalf("unexpected tidal config: %#v", config.TIDAL)
	}
	if config.HTTPTimeout != 45*time.Second {
		t.Fatalf("http timeout = %s, want 45s", config.HTTPTimeout)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.AppleMusicStorefront != "us" {
		t.Fatalf("apple music storefront = %q, want us", config.AppleMusicStorefront)
	}
	if config.ScoreWeights == (ScoreWeights{}) {
		t.Fatalf("expected default score weights")
	}
	if config.SongScoreWeights == (SongScoreWeights{}) {
		t.Fatalf("expected default song score weights")
	}
	if config.HTTPTimeout != 15*time.Second {
		t.Fatalf("http timeout = %s, want 15s", config.HTTPTimeout)
	}
}

func TestNormalizedConfigDefaultsSongWeights(t *testing.T) {
	config := normalizedConfig(Config{})
	if config.SongScoreWeights == (SongScoreWeights{}) {
		t.Fatalf("expected normalized config to include default song score weights")
	}
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
		if got := MatchStrengthForScore(tt.score); got != tt.want {
			t.Fatalf("MatchStrengthForScore(%d) = %q, want %q", tt.score, got, tt.want)
		}
	}
}

func TestNewWithAdaptersResolveAlbum(t *testing.T) {
	resolver := NewWithAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://fixture.test/source")
	if err != nil {
		t.Fatalf("ResolveAlbum error: %v", err)
	}
	if resolution.Source.Service != ServiceDeezer {
		t.Fatalf("source service = %q, want deezer", resolution.Source.Service)
	}
	match := resolution.Matches[ServiceSpotify]
	if match.Best == nil {
		t.Fatalf("expected spotify best match")
	}
	if match.Best.Candidate.CandidateID != "spotify-1" {
		t.Fatalf("candidate id = %q, want spotify-1", match.Best.Candidate.CandidateID)
	}
}

func TestNewWithEntityAdaptersResolveSong(t *testing.T) {
	resolver := NewWithEntityAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
		[]SongSourceAdapter{librarySongSourceAdapter{}},
		[]SongTargetAdapter{librarySongTargetAdapter{}},
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://fixture.test/songs/1")
	if err != nil {
		t.Fatalf("ResolveSong error: %v", err)
	}
	if resolution.Source.Service != ServiceSpotify {
		t.Fatalf("source service = %q, want spotify", resolution.Source.Service)
	}
	match := resolution.Matches[ServiceAppleMusic]
	if match.Best == nil {
		t.Fatalf("expected apple music best match")
	}
	if match.Best.Candidate.CandidateID != "apple-song-1" {
		t.Fatalf("candidate id = %q, want apple-song-1", match.Best.Candidate.CandidateID)
	}
}

func TestResolverResolveDispatchesByEntityType(t *testing.T) {
	resolver := NewWithEntityAdapters(
		[]SourceAdapter{librarySourceAdapter{}},
		[]TargetAdapter{libraryTargetAdapter{}},
		[]SongSourceAdapter{librarySongSourceAdapter{}},
		[]SongTargetAdapter{librarySongTargetAdapter{}},
	)

	albumEntity, err := resolver.Resolve(context.Background(), "https://fixture.test/source")
	if err != nil {
		t.Fatalf("Resolve album error: %v", err)
	}
	if albumEntity.Album == nil || albumEntity.Song != nil {
		t.Fatalf("expected album resolution only")
	}
	if albumEntity.Parsed.EntityType != "album" {
		t.Fatalf("parsed entity type = %q, want album", albumEntity.Parsed.EntityType)
	}

	songEntity, err := resolver.Resolve(context.Background(), "https://fixture.test/songs/1")
	if err != nil {
		t.Fatalf("Resolve song error: %v", err)
	}
	if songEntity.Song == nil || songEntity.Album != nil {
		t.Fatalf("expected song resolution only")
	}
	if songEntity.Parsed.EntityType != "song" {
		t.Fatalf("parsed entity type = %q, want song", songEntity.Parsed.EntityType)
	}
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
