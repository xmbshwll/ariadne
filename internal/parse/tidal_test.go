package parse

import "testing"

func TestTIDALAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical album url",
			raw:     "https://tidal.com/album/156205493",
			wantID:  "156205493",
			wantURL: "https://tidal.com/album/156205493",
		},
		{
			name:    "browse album url",
			raw:     "https://tidal.com/browse/album/156205493",
			wantID:  "156205493",
			wantURL: "https://tidal.com/album/156205493",
		},
		{
			name:    "www host with query string",
			raw:     "https://www.tidal.com/browse/album/156205493?u",
			wantID:  "156205493",
			wantURL: "https://tidal.com/album/156205493",
		},
		{
			name:    "listen host",
			raw:     "https://listen.tidal.com/album/156205493",
			wantID:  "156205493",
			wantURL: "https://tidal.com/album/156205493",
		},
		{
			name:    "wrong resource type",
			raw:     "https://tidal.com/track/123",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/album/156205493",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TIDALAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, "album", tt.wantID, tt.wantURL, "")
		})
	}
}

func TestTIDALSongURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical track url",
			raw:     "https://tidal.com/track/156205494",
			wantID:  "156205494",
			wantURL: "https://tidal.com/track/156205494",
		},
		{
			name:    "browse track url",
			raw:     "https://tidal.com/browse/track/156205494",
			wantID:  "156205494",
			wantURL: "https://tidal.com/track/156205494",
		},
		{
			name:    "listen host",
			raw:     "https://listen.tidal.com/track/156205494",
			wantID:  "156205494",
			wantURL: "https://tidal.com/track/156205494",
		},
		{
			name:    "www host with query string",
			raw:     "https://www.tidal.com/track/156205494?foo=bar",
			wantID:  "156205494",
			wantURL: "https://tidal.com/track/156205494",
		},
		{
			name:    "missing track id",
			raw:     "https://tidal.com/track",
			wantErr: true,
		},
		{
			name:    "wrong resource type",
			raw:     "https://tidal.com/album/156205493",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/track/156205494",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TIDALSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, "song", tt.wantID, tt.wantURL, "")
		})
	}
}
