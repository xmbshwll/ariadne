package normalize

import "strings"

// SearchQueries returns the Metadata Queries a single provider search should
// issue for a title and its artists: every title/artist variant pair first,
// then title-only fallbacks, deduplicated and in preference order.
func SearchQueries(title string, artists []string) []string {
	return FormattedSearchQueries(title, artists, func(titleVariant string, artistVariant string) string {
		return strings.Join([]string{titleVariant, artistVariant}, " ")
	}, func(titleVariant string) string {
		return titleVariant
	})
}

// FormattedSearchQueries builds Metadata Queries with a provider's own query
// syntax: formatWithArtist renders a title and artist pair, formatTitleOnly the
// fallback when a provider matches worse on a decorated query.
func FormattedSearchQueries(
	title string,
	artists []string,
	formatWithArtist func(string, string) string,
	formatTitleOnly func(string) string,
) []string {
	if strings.TrimSpace(title) == "" {
		return nil
	}

	queries := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)
	appendUnique := func(query string) {
		query = strings.TrimSpace(query)
		if query == "" {
			return
		}
		key := Text(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}

	titleVariants := SearchTitleVariants(title)
	artistVariants := SearchArtistVariants(artists)
	for _, titleVariant := range titleVariants {
		for _, artistVariant := range artistVariants {
			appendUnique(formatWithArtist(titleVariant, artistVariant))
		}
	}
	for _, titleVariant := range titleVariants {
		appendUnique(formatTitleOnly(titleVariant))
	}
	return queries
}

// SearchPrimaryQuery is the Metadata Query for providers that issue one search
// per Entity: the head of SearchQueries.
func SearchPrimaryQuery(title string, artists []string) string {
	queries := SearchQueries(title, artists)
	if len(queries) == 0 {
		return ""
	}
	return queries[0]
}

// NonEmpty drops blank-after-trim values, e.g. the ISRCs collected from tracks
// before they go into a search.
func NonEmpty(values []string) []string {
	trimmed := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}
