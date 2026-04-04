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
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.ID != tt.wantID {
				t.Fatalf("id = %q, want %q", got.ID, tt.wantID)
			}
			if got.CanonicalURL != tt.wantURL {
				t.Fatalf("canonical url = %q, want %q", got.CanonicalURL, tt.wantURL)
			}
		})
	}
}
