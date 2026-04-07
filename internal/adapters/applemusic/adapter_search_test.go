package applemusic

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
		"Solid Static Musica Transonic + Mainliner",
		"Solid Static Musica Transonic",
		"Solid Static Mainliner",
		"Solid Static",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("metadataQueries() = %#v, want %#v", got, want)
	}
}
