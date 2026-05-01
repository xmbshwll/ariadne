package adapterutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetadataQueries(t *testing.T) {
	queries := MetadataQueries("ΘΕΛΗΜΑ (Thelema)", []string{"DECIPHER"})

	assert.Equal(t, []string{
		"ΘΕΛΗΜΑ (Thelema) DECIPHER",
		"Thelema DECIPHER",
		"ΘΕΛΗΜΑ DECIPHER",
		"ΘΕΛΗΜΑ (Thelema)",
		"Thelema",
		"ΘΕΛΗΜΑ",
	}, queries)
}

func TestMetadataQueriesDeduplicatesNormalizedQueries(t *testing.T) {
	queries := MetadataQueries("Solid Static", []string{"Mainliner", "mainliner"})

	assert.Equal(t, []string{
		"Solid Static Mainliner",
		"Solid Static",
	}, queries)
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
	assert.Equal(t, "Solid Static Musica Transonic", PrimaryMetadataQuery("Solid Static", []string{"Musica Transonic"}))
	assert.Equal(t, "", PrimaryMetadataQuery(" ", []string{"Musica Transonic"}))
}
