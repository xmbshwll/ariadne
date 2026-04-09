package parse

import "testing"

func TestYouTubeMusicAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "browse url",
			raw:     "https://music.youtube.com/browse/MPREb_tQfaWH32ovE",
			wantID:  "MPREb_tQfaWH32ovE",
			wantURL: "https://music.youtube.com/browse/MPREb_tQfaWH32ovE",
		},
		{
			name:    "playlist canonical url",
			raw:     "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4&si=test",
			wantID:  "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4",
			wantURL: "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4",
		},
		{
			name:    "non album path rejected",
			raw:     "https://music.youtube.com/watch?v=example",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YouTubeMusicAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "album", tt.wantID, tt.wantURL, "")
		})
	}
}

func TestYouTubeMusicSongURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "watch url",
			raw:     "https://music.youtube.com/watch?v=dQw4w9WgXcQ&list=RDAMVMdQw4w9WgXcQ",
			wantID:  "dQw4w9WgXcQ",
			wantURL: "https://music.youtube.com/watch?v=dQw4w9WgXcQ",
		},
		{
			name:    "playlist url rejected",
			raw:     "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := YouTubeMusicSongURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, err)
				return
			}
			requireParsedURL(t, got, err, "song", tt.wantID, tt.wantURL, "")
		})
	}
}
