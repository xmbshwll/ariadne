package spotify

import (
	"reflect"
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestMetadataQueries(t *testing.T) {
	album := model.CanonicalAlbum{
		Title:   "Solid Static",
		Artists: []string{"Musica Transonic + Mainliner"},
	}

	got := metadataQueries(album)
	want := []string{
		"album:Solid Static artist:Musica Transonic + Mainliner",
		"album:Solid Static artist:Musica Transonic",
		"album:Solid Static artist:Mainliner",
		"album:Solid Static",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadataQueries() = %#v, want %#v", got, want)
	}
}
