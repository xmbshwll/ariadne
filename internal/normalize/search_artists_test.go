package normalize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchArtistVariants(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "keeps simple artist",
			input: []string{"Musica Transonic"},
			want:  []string{"Musica Transonic"},
		},
		{
			name:  "splits plus credit",
			input: []string{"Musica Transonic + Mainliner"},
			want:  []string{"Musica Transonic + Mainliner", "Musica Transonic", "Mainliner"},
		},
		{
			name:  "splits featuring credit",
			input: []string{"Musica Transonic feat. Mainliner"},
			want:  []string{"Musica Transonic feat. Mainliner", "Musica Transonic", "Mainliner"},
		},
		{
			name:  "deduplicates equivalent values",
			input: []string{"Musica Transonic + Mainliner", "musica transonic"},
			want:  []string{"Musica Transonic + Mainliner", "musica transonic", "Mainliner"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SearchArtistVariants(tt.input))
		})
	}
}
