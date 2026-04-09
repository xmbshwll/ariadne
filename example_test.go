package ariadne_test

import (
	"context"
	"fmt"
	"net/http"

	"github.com/xmbshwll/ariadne"
)

func ExampleDefaultConfig() {
	cfg := ariadne.DefaultConfig()

	fmt.Println(cfg.AppleMusicStorefront)
	fmt.Println(cfg.SpotifyEnabled())
	fmt.Println(cfg.TIDALEnabled())
	// Output:
	// us
	// false
	// false
}

func ExampleLoadConfigFromEnv() {
	cfg := ariadne.LoadConfigFromEnv(func(key string) string {
		switch key {
		case "APPLE_MUSIC_STOREFRONT":
			return "GB"
		case "SPOTIFY_CLIENT_ID":
			return "spotify-client"
		case "SPOTIFY_CLIENT_SECRET":
			return "spotify-secret"
		case "TIDAL_CLIENT_ID":
			return "tidal-client"
		case "TIDAL_CLIENT_SECRET":
			return "tidal-secret"
		default:
			return ""
		}
	})

	fmt.Println(cfg.AppleMusicStorefront)
	fmt.Println(cfg.SpotifyEnabled())
	fmt.Println(cfg.TIDALEnabled())
	// Output:
	// gb
	// true
	// true
}

func ExampleConfig_targetServices() {
	cfg := ariadne.DefaultConfig()
	cfg.TargetServices = []ariadne.ServiceName{ariadne.ServiceSpotify, ariadne.ServiceAppleMusic}

	fmt.Println(cfg.TargetServices[0])
	fmt.Println(cfg.TargetServices[1])
	// Output:
	// spotify
	// appleMusic
}

func ExampleMatchStrengthForScore() {
	fmt.Println(ariadne.MatchStrengthForScore(55))
	fmt.Println(ariadne.MatchStrengthForScore(85))
	// Output:
	// weak
	// probable
}

func ExampleNewWithClient() {
	resolver := ariadne.NewWithClient(&http.Client{}, ariadne.DefaultConfig())

	fmt.Println(resolver != nil)
	// Output:
	// true
}

func ExampleResolver_ResolveAlbum() {
	resolver := ariadne.NewWithAdapters(
		[]ariadne.SourceAdapter{exampleSourceAdapter{}},
		[]ariadne.TargetAdapter{exampleTargetAdapter{}},
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://example.test/albums/1")
	if err != nil {
		panic(err)
	}

	fmt.Println(resolution.Source.Title)
	fmt.Println(resolution.Matches[ariadne.ServiceSpotify].Best.URL)
	// Output:
	// Example Album
	// https://open.spotify.com/album/example-1
}

func ExampleResolver_ResolveSong() {
	resolver := ariadne.NewWithEntityAdapters(
		nil,
		nil,
		[]ariadne.SongSourceAdapter{exampleSongSourceAdapter{}},
		[]ariadne.SongTargetAdapter{exampleSongTargetAdapter{}},
	)

	resolution, err := resolver.ResolveSong(context.Background(), "https://example.test/songs/1")
	if err != nil {
		panic(err)
	}

	fmt.Println(resolution.Source.Title)
	fmt.Println(resolution.Matches[ariadne.ServiceAppleMusic].Best.URL)
	// Output:
	// Example Song
	// https://music.apple.com/us/album/example-album/2?i=3
}

func ExampleResolver_Resolve() {
	resolver := ariadne.NewWithEntityAdapters(
		[]ariadne.SourceAdapter{exampleSourceAdapter{}},
		[]ariadne.TargetAdapter{exampleTargetAdapter{}},
		[]ariadne.SongSourceAdapter{exampleSongSourceAdapter{}},
		[]ariadne.SongTargetAdapter{exampleSongTargetAdapter{}},
	)

	resolution, err := resolver.Resolve(context.Background(), "https://example.test/songs/1")
	if err != nil {
		panic(err)
	}

	fmt.Println(resolution.Parsed.EntityType)
	fmt.Println(resolution.Song.Source.Title)
	// Output:
	// song
	// Example Song
}

func ExampleNewWithAdaptersAndWeights() {
	weights := ariadne.DefaultScoreWeights()
	weights.TrackTitleStrong = 40

	resolver := ariadne.NewWithAdaptersAndWeights(
		[]ariadne.SourceAdapter{exampleSourceAdapter{}},
		[]ariadne.TargetAdapter{exampleTargetAdapter{}},
		weights,
	)

	resolution, err := resolver.ResolveAlbum(context.Background(), "https://example.test/albums/1")
	if err != nil {
		panic(err)
	}

	fmt.Println(resolution.Matches[ariadne.ServiceSpotify].Best.URL)
	// Output:
	// https://open.spotify.com/album/example-1
}

type exampleSourceAdapter struct{}

func (exampleSourceAdapter) Service() ariadne.ServiceName {
	return ariadne.ServiceDeezer
}

func (exampleSourceAdapter) ParseAlbumURL(raw string) (*ariadne.ParsedAlbumURL, error) {
	return &ariadne.ParsedAlbumURL{
		Service:      ariadne.ServiceDeezer,
		EntityType:   "album",
		ID:           "example-1",
		CanonicalURL: raw,
		RawURL:       raw,
	}, nil
}

func (exampleSourceAdapter) FetchAlbum(_ context.Context, parsed ariadne.ParsedAlbumURL) (*ariadne.CanonicalAlbum, error) {
	return &ariadne.CanonicalAlbum{
		Service:           parsed.Service,
		SourceID:          parsed.ID,
		SourceURL:         parsed.CanonicalURL,
		Title:             "Example Album",
		NormalizedTitle:   "example album",
		Artists:           []string{"Example Artist"},
		NormalizedArtists: []string{"example artist"},
		UPC:               "123456789012",
		Tracks: []ariadne.CanonicalTrack{
			{Title: "Intro", NormalizedTitle: "intro", ISRC: "ISRC0001"},
		},
	}, nil
}

type exampleTargetAdapter struct{}

func (exampleTargetAdapter) Service() ariadne.ServiceName {
	return ariadne.ServiceSpotify
}

func (exampleTargetAdapter) SearchByUPC(_ context.Context, upc string) ([]ariadne.CandidateAlbum, error) {
	return []ariadne.CandidateAlbum{{
		CanonicalAlbum: ariadne.CanonicalAlbum{
			Service:           ariadne.ServiceSpotify,
			SourceID:          "example-1",
			SourceURL:         "https://open.spotify.com/album/example-1",
			Title:             "Example Album",
			NormalizedTitle:   "example album",
			Artists:           []string{"Example Artist"},
			NormalizedArtists: []string{"example artist"},
			UPC:               upc,
		},
		CandidateID: "example-1",
		MatchURL:    "https://open.spotify.com/album/example-1",
	}}, nil
}

func (exampleTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]ariadne.CandidateAlbum, error) {
	return nil, nil
}

func (exampleTargetAdapter) SearchByMetadata(_ context.Context, _ ariadne.CanonicalAlbum) ([]ariadne.CandidateAlbum, error) {
	return nil, nil
}

type exampleSongSourceAdapter struct{}

func (exampleSongSourceAdapter) Service() ariadne.ServiceName {
	return ariadne.ServiceSpotify
}

func (exampleSongSourceAdapter) ParseSongURL(raw string) (*ariadne.ParsedURL, error) {
	return &ariadne.ParsedURL{
		Service:      ariadne.ServiceSpotify,
		EntityType:   "song",
		ID:           "song-1",
		CanonicalURL: raw,
		RawURL:       raw,
	}, nil
}

func (exampleSongSourceAdapter) FetchSong(_ context.Context, parsed ariadne.ParsedURL) (*ariadne.CanonicalSong, error) {
	return &ariadne.CanonicalSong{
		Service:              parsed.Service,
		SourceID:             parsed.ID,
		SourceURL:            parsed.CanonicalURL,
		Title:                "Example Song",
		NormalizedTitle:      "example song",
		Artists:              []string{"Example Artist"},
		NormalizedArtists:    []string{"example artist"},
		DurationMS:           180000,
		ISRC:                 "ISRCSONG001",
		TrackNumber:          1,
		AlbumTitle:           "Example Album",
		AlbumNormalizedTitle: "example album",
	}, nil
}

type exampleSongTargetAdapter struct{}

func (exampleSongTargetAdapter) Service() ariadne.ServiceName {
	return ariadne.ServiceAppleMusic
}

func (exampleSongTargetAdapter) SearchSongByISRC(_ context.Context, isrc string) ([]ariadne.CandidateSong, error) {
	return []ariadne.CandidateSong{{
		CanonicalSong: ariadne.CanonicalSong{
			Service:              ariadne.ServiceAppleMusic,
			SourceID:             "apple-song-1",
			SourceURL:            "https://music.apple.com/us/album/example-album/2?i=3",
			Title:                "Example Song",
			NormalizedTitle:      "example song",
			Artists:              []string{"Example Artist"},
			NormalizedArtists:    []string{"example artist"},
			DurationMS:           180050,
			ISRC:                 isrc,
			TrackNumber:          1,
			AlbumTitle:           "Example Album",
			AlbumNormalizedTitle: "example album",
		},
		CandidateID: "apple-song-1",
		MatchURL:    "https://music.apple.com/us/album/example-album/2?i=3",
	}}, nil
}

func (exampleSongTargetAdapter) SearchSongByMetadata(_ context.Context, _ ariadne.CanonicalSong) ([]ariadne.CandidateSong, error) {
	return nil, nil
}
