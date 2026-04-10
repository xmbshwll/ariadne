package parse

import (
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestAmazonMusicAlbumURL(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantID  string
		wantURL string
		wantErr bool
	}{
		{
			name:    "canonical album url",
			raw:     "https://music.amazon.com/albums/B0064UPU4G",
			wantID:  "B0064UPU4G",
			wantURL: "https://music.amazon.com/albums/B0064UPU4G",
		},
		{
			name:    "album url with query string",
			raw:     "https://music.amazon.com/albums/B0064UPU4G?ref=dm_sh_test",
			wantID:  "B0064UPU4G",
			wantURL: "https://music.amazon.com/albums/B0064UPU4G",
		},
		{
			name:    "wrong resource type",
			raw:     "https://music.amazon.com/artists/B0064UPU4G",
			wantErr: true,
		},
		{
			name:    "wrong host",
			raw:     "https://amazon.com/albums/B0064UPU4G",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AmazonMusicAlbumURL(tt.raw)
			if tt.wantErr {
				requireParseError(t, got, err)
				return
			}
			requireParsedURL(t, got, err, model.ServiceAmazonMusic, "album", tt.wantID, tt.wantURL, "")
		})
	}
}
