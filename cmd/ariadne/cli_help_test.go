package main

// White-box: the help functions assemble the resolve help text; the fixtures
// are catalog facts, asserted here so the rendered help cannot silently drift
// from what the Provider Catalog decides.

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelpServiceCaveats(t *testing.T) {
	caveats := helpServiceCaveats()

	tests := []struct {
		name        string
		wantSub     string
		wantMissing string
	}{
		{
			name:    "spotify names its missing Credential Token",
			wantSub: "spotify target search requires credentials: SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set.",
		},
		{
			name:    "tidal names its missing Credential Token",
			wantSub: "tidal target search requires credentials: TIDAL_CLIENT_ID and TIDAL_CLIENT_SECRET must be set.",
		},
		{
			name:        "services without caveats are absent",
			wantMissing: "appleMusic",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, caveats, tt.wantSub, tt.name)
			}
			if tt.wantMissing != "" {
				assert.NotContains(t, caveats, tt.wantMissing, tt.name)
			}
		})
	}
}

func TestHelpSongHydrationNote(t *testing.T) {
	note := helpSongHydrationNote()

	tests := []struct {
		name    string
		wantSub string
		missing []string
	}{
		{
			name:    "lists the catalog's song targets",
			wantSub: "  - Song resolution currently hydrates appleMusic, bandcamp, deezer, soundcloud, spotify, tidal.",
		},
		{
			name:    "omits services without song hydration",
			missing: []string{"amazonMusic", "youtubeMusic"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, note, tt.wantSub, tt.name)
			}
			for _, absent := range tt.missing {
				assert.NotContains(t, note, " "+absent+",", tt.name)
				assert.NotContains(t, note, " "+absent+".", tt.name)
			}
		})
	}
}

func TestHelpServiceNotes(t *testing.T) {
	notes := strings.Join(helpServiceNotes(), "\n")

	tests := []struct {
		name       string
		wantSub    string
		wantAbsent string
	}{
		{
			name:    "defers youtube music hydration",
			wantSub: "  - youtubeMusic song URLs are recognized, but runtime hydration remains deferred.",
		},
		{
			name:    "defers amazon music hydration",
			wantSub: "  - amazonMusic song URLs are recognized, but runtime hydration remains deferred.",
		},
		{
			name:       "does not defer a song target",
			wantAbsent: "spotify song URLs are recognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.wantSub != "" {
				assert.Contains(t, notes, tt.wantSub, tt.name)
			}
			if tt.wantAbsent != "" {
				assert.NotContains(t, notes, tt.wantAbsent, tt.name)
			}
		})
	}
}
