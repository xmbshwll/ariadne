package adapterutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataQueries(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		artists []string
		want    []string
	}{
		{
			name:    "transliterated title variants",
			title:   "ΘΕΛΗΜΑ (Thelema)",
			artists: []string{"DECIPHER"},
			want: []string{
				"ΘΕΛΗΜΑ (Thelema) DECIPHER",
				"Thelema DECIPHER",
				"ΘΕΛΗΜΑ DECIPHER",
				"ΘΕΛΗΜΑ (Thelema)",
				"Thelema",
				"ΘΕΛΗΜΑ",
			},
		},
		{
			name:    "deduplicates normalized queries",
			title:   "Solid Static",
			artists: []string{"Mainliner", "mainliner"},
			want: []string{
				"Solid Static Mainliner",
				"Solid Static",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, MetadataQueries(tt.title, tt.artists))
		})
	}
}

func TestFormattedMetadataQueries(t *testing.T) {
	queries := FormattedMetadataQueries("Solid Static", []string{"Musica Transonic + Mainliner"}, func(titleVariant string, artistVariant string) string {
		return titleVariant + " by " + artistVariant
	}, func(titleVariant string) string {
		return "title only " + titleVariant
	})

	assert.Equal(t, []string{
		"Solid Static by Musica Transonic + Mainliner",
		"Solid Static by Musica Transonic",
		"Solid Static by Mainliner",
		"title only Solid Static",
	}, queries)
}

func TestPrimaryMetadataQuery(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		artists []string
		want    string
	}{
		{name: "title and artist", title: "Solid Static", artists: []string{"Musica Transonic"}, want: "Solid Static Musica Transonic"},
		{name: "blank title yields no query", title: " ", artists: []string{"Musica Transonic"}, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PrimaryMetadataQuery(tt.title, tt.artists))
		})
	}
}
