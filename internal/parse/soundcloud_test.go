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
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "set", tt.wantID, tt.wantURL, "")
		})
	}
}

func TestSoundCloudSongURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical track url",
			raw:     "https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
			wantID:  "evidence-official/the-liner-notes-feat-aloe-1",
			wantURL: "https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
		},
		{
			name:    "www host with query string",
			raw:     "https://www.soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1?utm_source=test",
			wantID:  "evidence-official/the-liner-notes-feat-aloe-1",
			wantURL: "https://soundcloud.com/evidence-official/the-liner-notes-feat-aloe-1",
		},
		{
			name:    "set url is rejected",
			raw:     "https://soundcloud.com/evidence-official/sets/cats-dogs-6",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SoundCloudSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "song", tt.wantID, tt.wantURL, "")
		})
	}
}
