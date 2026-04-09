package spotify

import (
	"reflect"
	"testing"

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
				"album:Solid Static artist:Musica Transonic + Mainliner",
				"album:Solid Static artist:Musica Transonic",
				"album:Solid Static artist:Mainliner",
				"album:Solid Static",
			},
		},
		{
			name: "includes alternate title fallbacks",
			album: model.CanonicalAlbum{
				Title:   "ΘΕΛΗΜΑ (Thelema)",
				Artists: []string{"DECIPHER"},
			},
			want: []string{
				"album:ΘΕΛΗΜΑ (Thelema) artist:DECIPHER",
				"album:ΘΕΛΗΜΑ (Thelema)",
				"album:Thelema artist:DECIPHER",
				"album:Thelema",
				"album:ΘΕΛΗΜΑ artist:DECIPHER",
				"album:ΘΕΛΗΜΑ",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := metadataQueries(tt.album)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("metadataQueries() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestSongMetadataQueries(t *testing.T) {
	song := model.CanonicalSong{
		Title:   "ΘΕΛΗΜΑ (Thelema)",
		Artists: []string{"DECIPHER"},
	}

	got := songMetadataQueries(song)
	want := []string{
		"track:ΘΕΛΗΜΑ (Thelema) artist:DECIPHER",
		"track:ΘΕΛΗΜΑ (Thelema)",
		"track:Thelema artist:DECIPHER",
		"track:Thelema",
		"track:ΘΕΛΗΜΑ artist:DECIPHER",
		"track:ΘΕΛΗΜΑ",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("songMetadataQueries() = %#v, want %#v", got, want)
	}
}
