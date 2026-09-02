package canonical_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/canonical"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{name: "returns the first non-empty value", values: []string{"", "first", "second"}, want: "first"},
		{name: "whitespace-only values count as absent", values: []string{"  ", "\t", "value"}, want: "value"},
		{name: "values are trimmed", values: []string{" padded "}, want: "padded"},
		{name: "all empty answers empty", values: []string{"", "  "}, want: ""},
		{name: "no values answers empty", values: nil, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonical.FirstNonEmpty(tt.values...), tt.name)
		})
	}
}

func TestSingleArtistList(t *testing.T) {
	tests := []struct {
		name   string
		artist string
		want   []string
	}{
		{name: "one artist becomes a one-element list", artist: "The Beatles", want: []string{"The Beatles"}},
		{name: "padded artist is trimmed", artist: "  The Beatles  ", want: []string{"The Beatles"}},
		{name: "blank artist becomes nil", artist: "   ", want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonical.SingleArtistList(tt.artist), tt.name)
		})
	}
}

func TestDateOnly(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{name: "a full timestamp truncates to its date", value: "1969-09-26T07:00:00Z", want: "1969-09-26"},
		{name: "a date passes through", value: "1969-09-26", want: "1969-09-26"},
		{name: "a short value passes through unchanged", value: "1969-09", want: "1969-09"},
		{name: "padded values trim first", value: "  1969-09-26T07:00:00Z  ", want: "1969-09-26"},
		{name: "empty stays empty", value: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonical.DateOnly(tt.value), tt.name)
		})
	}
}

func TestCandidateAlbumWrapsCanonicalAlbum(t *testing.T) {
	album := model.CanonicalAlbum{Service: model.ServiceDeezer, SourceID: "12047952", SourceURL: "https://www.deezer.com/album/12047952"}

	candidate := canonical.CandidateAlbum(album)

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "candidate id is the source id", got: candidate.CandidateID, want: "12047952"},
		{name: "match url is the source url", got: candidate.MatchURL, want: album.SourceURL},
		{name: "canonical album is embedded", got: candidate.CanonicalAlbum, want: album},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got, tt.name)
		})
	}
}

func TestCandidateSongWrapsCanonicalSong(t *testing.T) {
	song := model.CanonicalSong{Service: model.ServiceSpotify, SourceID: "track-1", SourceURL: "https://open.spotify.com/track/track-1"}

	candidate := canonical.CandidateSong(song)

	tests := []struct {
		name string
		got  any
		want any
	}{
		{name: "candidate id is the source id", got: candidate.CandidateID, want: "track-1"},
		{name: "match url is the source url", got: candidate.MatchURL, want: song.SourceURL},
		{name: "canonical song is embedded", got: candidate.CanonicalSong, want: song},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got, tt.name)
		})
	}
}

func TestISODurationMilliseconds(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "hours minutes and seconds sum", value: "PT1H2M3S", want: 3723000},
		{name: "fractional seconds parse", value: "PT1.5S", want: 1500},
		{name: "lowercase units parse", value: "pt1h", want: 3600000},
		{name: "minutes beyond sixty are kept, not wrapped", value: "PT90M", want: 5400000},
		{name: "empty duration answers zero", value: "", want: 0},
		{name: "unparseable duration answers zero", value: "not-a-duration", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, canonical.ISODurationMilliseconds(tt.value), tt.name)
		})
	}
}
