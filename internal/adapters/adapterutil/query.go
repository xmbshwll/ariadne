package adapterutil

import "strings"

func TitleAndFirstArtistQuery(title string, artists []string) string {
	parts := make([]string, 0, 2)
	if title = strings.TrimSpace(title); title != "" {
		parts = append(parts, title)
	}
	for _, artist := range artists {
		artist = strings.TrimSpace(artist)
		if artist == "" {
			continue
		}
		parts = append(parts, artist)
		break
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
