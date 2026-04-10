package parse

import (
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestDeezerAlbumURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantID     string
		wantURL    string
		wantRegion string
		wantErr    bool
	}{
		{
			name:       "canonical",
			raw:        "https://www.deezer.com/album/12047952",
			wantID:     "12047952",
			wantURL:    "https://www.deezer.com/album/12047952",
			wantRegion: "",
		},
		{
			name:       "region and query string",
			raw:        "https://www.deezer.com/us/album/12047952?utm_source=test",
			wantID:     "12047952",
			wantURL:    "https://www.deezer.com/album/12047952",
			wantRegion: "us",
		},
		{
			name:    "missing album id",
			raw:     "https://www.deezer.com/album",
			wantErr: true,
		},
		{
			name:    "non album url",
			raw:     "https://www.deezer.com/track/116348452",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/album/12047952",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeezerAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, model.ServiceDeezer, "album", tt.wantID, tt.wantURL, tt.wantRegion)
		})
	}
}

func TestDeezerSongURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantID     string
		wantURL    string
		wantRegion string
		wantErr    bool
	}{
		{
			name:       "canonical",
			raw:        "https://www.deezer.com/track/116348452",
			wantID:     "116348452",
			wantURL:    "https://www.deezer.com/track/116348452",
			wantRegion: "",
		},
		{
			name:       "region and query string",
			raw:        "https://www.deezer.com/us/track/116348452?utm_source=test",
			wantID:     "116348452",
			wantURL:    "https://www.deezer.com/track/116348452",
			wantRegion: "us",
		},
		{
			name:    "missing track id",
			raw:     "https://www.deezer.com/track",
			wantErr: true,
		},
		{
			name:    "non song url",
			raw:     "https://www.deezer.com/album/12047952",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/track/116348452",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DeezerSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, model.ServiceDeezer, "song", tt.wantID, tt.wantURL, tt.wantRegion)
		})
	}
}
