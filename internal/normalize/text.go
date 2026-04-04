package normalize

import (
	"regexp"
	"strings"
)

var nonAlnum = regexp.MustCompile(`[^\p{L}\p{N}]+`)

// Text normalizes a free-form metadata string for matching.
func Text(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "&", " and ")
	s = nonAlnum.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

// Artists normalizes a list of artist names for matching.
func Artists(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		normalized := Text(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}
