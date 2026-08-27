package resolve_test

import (
	"context"
	"errors"
	"testing"

	resolve "github.com/xmbshwll/ariadne/internal/resolve"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/adapters"
	"github.com/xmbshwll/ariadne/internal/adapters/base"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errSourceInputTest = errors.New("source input test")

// sourceInputTestAdapter is the bare minimum an Adapter can be: it names a
// service and answers every capability with ErrUnsupported, which is what
// Source Input recognition needs to see.
type sourceInputTestAdapter struct {
	base.Unsupported
	service model.ServiceName
}

func (a sourceInputTestAdapter) Service() model.ServiceName {
	return a.service
}

func (sourceInputTestAdapter) Capabilities() adapters.Capabilities {
	return adapters.Capabilities{}
}

type fatalSourceInputError struct {
	error
}

func (fatalSourceInputError) FatalParseFailure() bool {
	return true
}

func (e fatalSourceInputError) Unwrap() error {
	return e.error
}

func TestResolveSourceInputHydratesRecognizedSource(t *testing.T) {
	parsed, adapter, err := resolve.RecognizeSourceInput(
		[]adapters.Adapter{sourceInputTestAdapter{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(adapters.Adapter) (*model.ParsedURL, error) {
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "1"}, nil
		},
	)
	require.NoError(t, err)

	album, err := resolve.HydrateSourceInput(
		context.Background(),
		adapter,
		"album",
		resolve.ErrNilSourceAlbum,
		func(context.Context) (*model.CanonicalAlbum, error) {
			return &model.CanonicalAlbum{Service: adapter.Service(), SourceID: parsed.ID}, nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, "1", parsed.ID)
	assert.Equal(t, "1", album.SourceID)
}

func TestResolveSourceInputPreservesFatalParseFailure(t *testing.T) {
	_, _, err := resolve.RecognizeSourceInput(
		[]adapters.Adapter{sourceInputTestAdapter{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(adapters.Adapter) (*model.ParsedURL, error) {
			return nil, fatalSourceInputError{errSourceInputTest}
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errSourceInputTest)
}

func TestResolveSourceInputReportsNilHydration(t *testing.T) {
	_, adapter, err := resolve.RecognizeSourceInput(
		[]adapters.Adapter{sourceInputTestAdapter{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(adapters.Adapter) (*model.ParsedURL, error) {
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: model.EntityTypeAlbum, ID: "1"}, nil
		},
	)
	require.NoError(t, err)

	_, err = resolve.HydrateSourceInput(
		context.Background(),
		adapter,
		"album",
		resolve.ErrNilSourceAlbum,
		func(context.Context) (*model.CanonicalAlbum, error) {
			return nil, nil //nolint:nilnil // Exercise source input nil hydration outcome.
		},
	)

	require.Error(t, err)
	assert.EqualError(t, err, "fetch source album returned nil from spotify")
	assert.ErrorIs(t, err, resolve.ErrNilSourceAlbum)
}
