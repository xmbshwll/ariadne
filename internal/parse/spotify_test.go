package parse

import "testing"

func TestSpotifyAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical",
			raw:     "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
			wantID:  "0ETFjACtuP2ADo6LFhL6HN",
			wantURL: "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
		},
		{
			name:    "query string",
			raw:     "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN?si=test",
			wantID:  "0ETFjACtuP2ADo6LFhL6HN",
			wantURL: "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
		},
		{
			name:    "wrong resource type",
			raw:     "https://open.spotify.com/track/2EqlS6tkEnglzr7tkKAAYD",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/album/0ETFjACtuP2ADo6LFhL6HN",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SpotifyAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "album", tt.wantID, tt.wantURL, "")
		})
	}
}

func TestSpotifySongURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical",
			raw:     "https://open.spotify.com/track/2EqlS6tkEnglzr7tkKAAYD",
			wantID:  "2EqlS6tkEnglzr7tkKAAYD",
			wantURL: "https://open.spotify.com/track/2EqlS6tkEnglzr7tkKAAYD",
		},
		{
			name:    "query string",
			raw:     "https://open.spotify.com/track/2EqlS6tkEnglzr7tkKAAYD?si=test",
			wantID:  "2EqlS6tkEnglzr7tkKAAYD",
			wantURL: "https://open.spotify.com/track/2EqlS6tkEnglzr7tkKAAYD",
		},
		{
			name:    "wrong resource type",
			raw:     "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/track/2EqlS6tkEnglzr7tkKAAYD",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SpotifySongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "song", tt.wantID, tt.wantURL, "")
		})
	}
}
