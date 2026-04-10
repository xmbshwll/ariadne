package normalize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchTitleVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "keeps simple title",
			input: "Solid Static",
			want:  []string{"Solid Static"},
		},
		{
			name:  "strips edition parenthetical",
			input: "Solid Static (Deluxe Edition)",
			want:  []string{"Solid Static (Deluxe Edition)", "Solid Static"},
		},
		{
			name:  "extracts latin alternate title from mixed-script title",
			input: "ΘΕΛΗΜΑ (Thelema)",
			want:  []string{"ΘΕΛΗΜΑ (Thelema)", "Thelema", "ΘΕΛΗΜΑ"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SearchTitleVariants(tt.input))
		})
	}
}
