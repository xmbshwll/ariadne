package adapterutil

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/normalize"
)

func TitleAndFirstArtistQuery(title string, artists []string) string {
	return PrimaryMetadataQuery(title, artists)
}

func PrimaryMetadataQuery(title string, artists []string) string {
	queries := MetadataQueries(title, artists)
	if len(queries) == 0 {
		return ""
	}
	return queries[0]
}

func MetadataQueries(title string, artists []string) []string {
	return FormattedMetadataQueries(title, artists, func(titleVariant string, artistVariant string) string {
		return strings.Join([]string{titleVariant, artistVariant}, " ")
	}, func(titleVariant string) string {
		return titleVariant
	})
}

func FormattedMetadataQueries(title string, artists []string, formatWithArtist func(string, string) string, formatTitleOnly func(string) string) []string {
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
		key := normalize.Text(query)
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		queries = append(queries, query)
	}

	titleVariants := normalize.SearchTitleVariants(title)
	artistVariants := normalize.SearchArtistVariants(artists)
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
