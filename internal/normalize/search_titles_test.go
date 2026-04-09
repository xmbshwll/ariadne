package normalize

import (
	"reflect"
	"testing"
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
			got := SearchTitleVariants(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("SearchTitleVariants(%q) = %#v, want %#v", tt.input, got, tt.want)
			}
		})
	}
}
