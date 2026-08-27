package main

import (
	"context"
	"errors"
	"testing"

	"github.com/xmbshwll/ariadne"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errUnsupportedCLIFixture = errors.New("unsupported")

// fixtureResolverForCLI stands in for ariadne.Resolver in CLI tests. The CLI's
// job is flags, config, output shapes, and error mapping, so a test describes
// the resolution it wants rendered instead of assembling a whole Entity
// Resolution pipeline; ariadne and internal/resolve tests cover ranking itself.
// A nil album or song means "this input is not that Entity Shape", which is how
// the auto-dispatch path decides.
type fixtureResolverForCLI struct {
	album *ariadne.Resolution
	song  *ariadne.SongResolution
	// albumErr and songErr stand in for a resolver that rejects the URL.
	albumErr error
	songErr  error
}

func (r fixtureResolverForCLI) ResolveAlbum(_ context.Context, _ string) (*ariadne.Resolution, error) {
	if r.albumErr != nil {
		return nil, r.albumErr
	}
	if r.album == nil {
		return nil, errUnsupportedCLIFixture
	}
	return r.album, nil
}

func (r fixtureResolverForCLI) ResolveSong(_ context.Context, _ string) (*ariadne.SongResolution, error) {
	if r.songErr != nil {
		return nil, r.songErr
	}
	if r.song == nil {
		return nil, errUnsupportedCLIFixture
	}
	return r.song, nil
}

// Resolve mirrors the dispatch order the real Resolver uses: song first, album
// when the input is not a song URL.
func (r fixtureResolverForCLI) Resolve(_ context.Context, _ string) (*ariadne.EntityResolution, error) {
	if r.song != nil {
		return &ariadne.EntityResolution{Parsed: r.song.Parsed, Song: r.song}, nil
	}
	if r.album != nil {
		return &ariadne.EntityResolution{Parsed: r.album.Parsed, Album: r.album}, nil
	}
	return nil, errUnsupportedCLIFixture
}

// fixtureAlbumResolution describes one album resolution for a source the CLI is
// asked to resolve: the source's own URL is the input URL.
func fixtureAlbumResolution(source ariadne.CanonicalAlbum, matches map[ariadne.ServiceName]ariadne.MatchResult) *ariadne.Resolution {
	return &ariadne.Resolution{
		InputURL: source.SourceURL,
		Parsed: ariadne.ParsedAlbumURL{
			Service: source.Service, EntityType: model.EntityTypeAlbum,
			ID: source.SourceID, CanonicalURL: source.SourceURL, RawURL: source.SourceURL,
		},
		Source:  source,
		Matches: matches,
	}
}

// fixtureSongResolution describes one song resolution for a source the CLI is
// asked to resolve.
func fixtureSongResolution(source ariadne.CanonicalSong, matches map[ariadne.ServiceName]ariadne.SongMatchResult) *ariadne.SongResolution {
	return &ariadne.SongResolution{
		InputURL: source.SourceURL,
		Parsed: ariadne.ParsedURL{
			Service: source.Service, EntityType: model.EntityTypeSong,
			ID: source.SourceID, CanonicalURL: source.SourceURL, RawURL: source.SourceURL,
		},
		Source:  source,
		Matches: matches,
	}
}

// withResolverFactory installs a resolver factory for one test.
func withResolverFactory(t *testing.T, factory func(ariadne.Config) entityResolver) {
	t.Helper()
	originalFactory := resolverFactory
	resolverFactory = factory
	t.Cleanup(func() {
		resolverFactory = originalFactory
	})
}

// withFixtureResolver installs a fixed set of resolutions for one test.
func withFixtureResolver(t *testing.T, fixture fixtureResolverForCLI) {
	t.Helper()
	withResolverFactory(t, func(ariadne.Config) entityResolver { return fixture })
}
