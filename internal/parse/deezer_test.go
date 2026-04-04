package parse

import "testing"

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
			if got.RegionHint != tt.wantRegion {
				t.Fatalf("region = %q, want %q", got.RegionHint, tt.wantRegion)
			}
		})
	}
}
