package parse

import "testing"

func TestSoundCloudAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical set url",
			raw:     "https://soundcloud.com/evidence-official/sets/cats-dogs-6",
			wantID:  "evidence-official/sets/cats-dogs-6",
			wantURL: "https://soundcloud.com/evidence-official/sets/cats-dogs-6",
		},
		{
			name:    "www host with query string",
			raw:     "https://www.soundcloud.com/evidence-official/sets/cats-dogs-6?utm_source=test",
			wantID:  "evidence-official/sets/cats-dogs-6",
			wantURL: "https://soundcloud.com/evidence-official/sets/cats-dogs-6",
		},
		{
			name:    "track url is rejected",
			raw:     "https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SoundCloudAlbumURL(tt.raw)
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
