package normalize

import (
	"regexp"
	"strings"
)

var artistSearchSplitPattern = regexp.MustCompile(`(?i)\s*(?:\+|&|,|\bfeat\.?\b|\bfeaturing\b|\bwith\b|\bx\b|\s/\s)\s*`)

// SearchArtistVariants returns ordered artist strings suitable for search
// fallbacks. It keeps the original artist credits first, then adds split-out
// artist components for compound credits such as "A + B" or "A feat. B".
func SearchArtistVariants(values []string) []string {
	variants := make([]string, 0, len(values)*2)
	seen := make(map[string]struct{}, len(values)*2)

	appendUnique := func(value string) {
		value = strings.Trim(strings.TrimSpace(value), ".")
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := Text(value)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		variants = append(variants, value)
	}

	for _, value := range values {
		appendUnique(value)
	}
	for _, value := range values {
		for _, part := range artistSearchSplitPattern.Split(value, -1) {
			appendUnique(part)
		}
	}

	return variants
}
