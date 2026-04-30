package resolve

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errSourceInputTest = errors.New("source input test")

type sourceInputTestAdapter struct {
	service model.ServiceName
}

func (a sourceInputTestAdapter) Service() model.ServiceName {
	return a.service
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
	result, err := resolveSourceInput(
		context.Background(),
		[]sourceInputTestAdapter{{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(sourceInputTestAdapter, string) (*model.ParsedURL, error) {
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "album", ID: "1"}, nil
		},
		func(_ context.Context, source sourceInputTestAdapter, parsed model.ParsedURL) (*model.CanonicalAlbum, error) {
			return &model.CanonicalAlbum{Service: source.Service(), SourceID: parsed.ID}, nil
		},
		"album",
		errNilSourceAlbum,
	)

	require.NoError(t, err)
	assert.Equal(t, "1", result.Parsed.ID)
	assert.Equal(t, "1", result.Entity.SourceID)
}

func TestResolveSourceInputPreservesFatalParseFailure(t *testing.T) {
	hydrated := false

	_, err := resolveSourceInput(
		context.Background(),
		[]sourceInputTestAdapter{{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(sourceInputTestAdapter, string) (*model.ParsedURL, error) {
			return nil, fatalSourceInputError{errSourceInputTest}
		},
		func(context.Context, sourceInputTestAdapter, model.ParsedURL) (*model.CanonicalAlbum, error) {
			hydrated = true
			return nil, nil //nolint:nilnil // Hydration must not run when parse failure is fatal.
		},
		"album",
		errNilSourceAlbum,
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, errSourceInputTest)
	assert.False(t, hydrated)
}

func TestResolveSourceInputReportsNilHydration(t *testing.T) {
	_, err := resolveSourceInput(
		context.Background(),
		[]sourceInputTestAdapter{{service: model.ServiceSpotify}},
		"https://open.spotify.com/album/1",
		func(sourceInputTestAdapter, string) (*model.ParsedURL, error) {
			return &model.ParsedURL{Service: model.ServiceSpotify, EntityType: "album", ID: "1"}, nil
		},
		func(context.Context, sourceInputTestAdapter, model.ParsedURL) (*model.CanonicalAlbum, error) {
			return nil, nil //nolint:nilnil // Exercise source input nil hydration outcome.
		},
		"album",
		errNilSourceAlbum,
	)

	require.Error(t, err)
	assert.EqualError(t, err, "fetch source album returned nil from spotify")
	assert.ErrorIs(t, err, errNilSourceAlbum)
}
