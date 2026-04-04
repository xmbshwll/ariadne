package parse

import "testing"

func TestAppleMusicAlbumURL(t *testing.T) {
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
			raw:        "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
			wantID:     "1441164426",
			wantURL:    "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
			wantRegion: "us",
		},
		{
			name:       "query string",
			raw:        "https://music.apple.com/us/album/abbey-road-remastered/1441164426?uo=4",
			wantID:     "1441164426",
			wantURL:    "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
			wantRegion: "us",
		},
		{
			name:    "wrong resource",
			raw:     "https://music.apple.com/us/artist/the-beatles/136975",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/us/album/abbey-road-remastered/1441164426",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AppleMusicAlbumURL(tt.raw)
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
