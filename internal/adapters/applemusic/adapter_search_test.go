package applemusic

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestMetadataQueries(t *testing.T) {
	tests := []struct {
		name  string
		album model.CanonicalAlbum
		want  []string
	}{
		{
			name: "includes artist fallbacks",
			album: model.CanonicalAlbum{
				Title:   "Solid Static",
				Artists: []string{"Musica Transonic + Mainliner"},
			},
			want: []string{
				"Solid Static Musica Transonic + Mainliner",
				"Solid Static Musica Transonic",
				"Solid Static Mainliner",
				"Solid Static",
			},
		},
		{
			name: "includes alternate title fallbacks",
			album: model.CanonicalAlbum{
				Title:   "ΘΕΛΗΜΑ (Thelema)",
				Artists: []string{"DECIPHER"},
			},
			want: []string{
				"ΘΕΛΗΜΑ (Thelema) DECIPHER",
				"ΘΕΛΗΜΑ (Thelema)",
				"Thelema DECIPHER",
				"Thelema",
				"ΘΕΛΗΜΑ DECIPHER",
				"ΘΕΛΗΜΑ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, metadataQueries(tt.album))
		})
	}
}
