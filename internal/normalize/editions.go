package normalize

import "strings"

var editionKeywords = []string{
	"deluxe",
	"remaster",
	"remastered",
	"anniversary",
	"live",
	"acoustic",
	"explicit",
	"clean",
	"mix",
}

// EditionHints extracts simple edition markers from an album title.
func EditionHints(title string) []string {
	lower := strings.ToLower(title)
	out := make([]string, 0, len(editionKeywords))
	for _, keyword := range editionKeywords {
		if strings.Contains(lower, keyword) {
			out = append(out, keyword)
		}
	}
	return out
}
