package parse

import "testing"

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BandcampAlbumURL(tt.raw)
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
