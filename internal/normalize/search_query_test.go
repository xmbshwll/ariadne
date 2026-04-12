package normalize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchPrimaryQuery(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		artists []string
		want    string
	}{
		{
			name:    "prefers title and artist",
			title:   "Solid Static (Deluxe Edition)",
			artists: []string{"Musica Transonic + Mainliner"},
			want:    "Solid Static (Deluxe Edition) Musica Transonic + Mainliner",
		},
		{
			name:  "falls back to title when artists missing",
			title: "Shadows among trees",
			want:  "Shadows among trees",
		},
		{
			name:    "uses stripped title when original is empty after trimming",
			title:   " テスト (Test) ",
			artists: []string{"Artist"},
			want:    "テスト (Test) Artist",
		},
		{
			name: "empty title returns empty query",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SearchPrimaryQuery(tt.title, tt.artists))
		})
	}
}
