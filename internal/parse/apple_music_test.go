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
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "album", tt.wantID, tt.wantURL, tt.wantRegion)
		})
	}
}

func TestAppleMusicSongURL(t *testing.T) {
	tests := []struct {
		name       string
		raw        string
		wantID     string
		wantURL    string
		wantRegion string
		wantErr    bool
	}{
		{
			name:       "album page song reference",
			raw:        "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
			wantID:     "1441164430",
			wantURL:    "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
			wantRegion: "us",
		},
		{
			name:       "canonical url escapes track id query value",
			raw:        "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=track%2Bid",
			wantID:     "track+id",
			wantURL:    "https://music.apple.com/us/album/abbey-road-remastered/1441164426?i=track%2Bid",
			wantRegion: "us",
		},
		{
			name:    "missing track id",
			raw:     "https://music.apple.com/us/album/abbey-road-remastered/1441164426",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://example.com/us/album/abbey-road-remastered/1441164426?i=1441164430",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AppleMusicSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "song", tt.wantID, tt.wantURL, tt.wantRegion)
		})
	}
}
