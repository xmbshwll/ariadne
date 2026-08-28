package normalize

import "strings"

// EditionMarkers is Ariadne's one edition vocabulary: every marker word any
// layer recognizes. Consumers derive their subsets from it, so the words cannot
// drift between scoring and hint extraction. Longest markers come first, so a
// consumer that blanks matches as it scans (longest-match-first) sees
// "super deluxe" before "deluxe".
var EditionMarkers = []string{
	"super deluxe",
	"remastered",
	"anniversary",
	"acoustic",
	"explicit",
	"deluxe",
	"remaster",
	"clean",
	"live",
	"mix",
}

// EditionHints extracts simple edition markers from an album title. The
// classification set includes the content flags explicit and clean, which
// describe a release rather than edition text.
func EditionHints(title string) []string {
	lower := strings.ToLower(title)
	out := make([]string, 0, len(EditionMarkers))
	for _, keyword := range EditionMarkers {
		if strings.Contains(lower, keyword) {
			out = append(out, keyword)
		}
	}
	return out
}

// EditionComparisonMarkers returns the marker words that may be stripped from a
// title when two releases are compared for sameness: every marker except the
// content flags, which stay in the title so explicit/clean editions keep
// distinguishable names.
func EditionComparisonMarkers() []string {
	markers := make([]string, 0, len(EditionMarkers))
	for _, keyword := range EditionMarkers {
		if keyword == "explicit" || keyword == "clean" {
			continue
		}
		markers = append(markers, keyword)
	}
	return markers
}
