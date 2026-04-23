package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestLookup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want model.ServiceName
		ok   bool
	}{
		{name: "canonical name", raw: "spotify", want: model.ServiceSpotify, ok: true},
		{name: "hyphen alias", raw: "apple-music", want: model.ServiceAppleMusic, ok: true},
		{name: "underscore alias", raw: "yt_music", want: model.ServiceYouTubeMusic, ok: true},
		{name: "trimmed alias", raw: " amazon ", want: model.ServiceAmazonMusic, ok: true},
		{name: "unknown", raw: "napster"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, ok := Lookup(tt.raw)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, service)
		})
	}
}

func TestLookupTarget(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want model.ServiceName
		ok   bool
	}{
		{name: "spotify target", raw: "spotify", want: model.ServiceSpotify, ok: true},
		{name: "ytmusic alias", raw: "ytmusic", want: model.ServiceYouTubeMusic, ok: true},
		{name: "amazon excluded", raw: "amazon"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			service, ok := LookupTarget(tt.raw)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, service)
		})
	}
}

func TestAliasesFor(t *testing.T) {
	t.Parallel()

	aliases := AliasesFor(model.ServiceYouTubeMusic)
	require.Equal(t, []string{"youtubemusic", "ytmusic"}, aliases)

	aliases[0] = "mutated"
	assert.Equal(t, []string{"youtubemusic", "ytmusic"}, AliasesFor(model.ServiceYouTubeMusic))
	assert.Equal(t, []string{"spotify"}, AliasesFor(model.ServiceSpotify))
}
