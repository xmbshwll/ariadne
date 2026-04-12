package normalize

import "strings"

// SearchPrimaryQuery returns the first preferred metadata query used by
// single-query adapters. It follows the same ordering as the multi-query
// builders: title+artist combinations first, then title-only fallbacks.
func SearchPrimaryQuery(title string, artists []string) string {
	titleVariants := SearchTitleVariants(title)
	artistVariants := SearchArtistVariants(artists)

	for _, titleVariant := range titleVariants {
		titleVariant = strings.TrimSpace(titleVariant)
		if titleVariant == "" {
			continue
		}
		for _, artistVariant := range artistVariants {
			query := strings.TrimSpace(strings.Join([]string{titleVariant, artistVariant}, " "))
			if query != "" {
				return query
			}
		}
		return titleVariant
	}

	return ""
}
