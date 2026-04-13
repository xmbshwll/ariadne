package parse

import (
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestBandcampAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical",
			raw:     "https://comradiation.bandcamp.com/album/l-n-abaty-abbey-road",
			wantID:  "l-n-abaty-abbey-road",
			wantURL: "https://comradiation.bandcamp.com/album/l-n-abaty-abbey-road",
		},
		{
			name:    "query string",
			raw:     "https://comradiation.bandcamp.com/album/l-n-abaty-abbey-road?from=search",
			wantID:  "l-n-abaty-abbey-road",
			wantURL: "https://comradiation.bandcamp.com/album/l-n-abaty-abbey-road",
		},
		{
			name:    "wrong path",
			raw:     "https://comradiation.bandcamp.com/track/example",
			wantErr: true,
		},
		{
			name:    "unsupported host",
			raw:     "https://open.spotify.com/album/example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BandcampAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, model.ServiceBandcamp, "album", tt.wantID, tt.wantURL, "")
		})
	}
}

func TestBandcampSongURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical",
			raw:     "https://comradiation.bandcamp.com/track/come-together",
			wantID:  "come-together",
			wantURL: "https://comradiation.bandcamp.com/track/come-together",
		},
		{
			name:    "query string",
			raw:     "https://comradiation.bandcamp.com/track/come-together?from=search",
			wantID:  "come-together",
			wantURL: "https://comradiation.bandcamp.com/track/come-together",
		},
		{
			name:    "wrong path",
			raw:     "https://comradiation.bandcamp.com/album/l-n-abaty-abbey-road",
			wantErr: true,
		},
		{
			name:    "unsupported host",
			raw:     "https://open.spotify.com/track/example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BandcampSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, model.ServiceBandcamp, "song", tt.wantID, tt.wantURL, "")
		})
	}
}
