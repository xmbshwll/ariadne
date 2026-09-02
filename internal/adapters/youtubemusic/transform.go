package youtubemusic

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/xmbshwll/ariadne/internal/canonical"
	"github.com/xmbshwll/ariadne/internal/htmlx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

type searchCandidate struct {
	Title    string
	BrowseID string
	Artist   string
}

var trackPlayCountPattern = regexp.MustCompile(`\b\d[\d,._\s]*\s*(views|wiedergaben)\b`)

func extractAlbum(body []byte, fallbackURL string) (*model.CanonicalAlbum, error) {
	canonicalURL := extractFirstGroup(canonicalURLPattern, body)
	if canonicalURL == "" {
		canonicalURL = strings.TrimSpace(fallbackURL)
	}
	title := cleanAlbumTitle(extractFirstGroup(ogTitlePattern, body))
	if title == "" {
		return nil, errors.Join(ErrMalformedYouTubeMusicPage, errYouTubeMusicAlbumTitleNotFound)
	}

	artist := html.UnescapeString(extractFirstGroup(subtitleArtistPattern, body))
	trackTitles := ExtractTrackTitles(body)
	artists := canonical.SingleArtistList(artist)
	sourceID := youTubeMusicAlbumSourceID(canonicalURL)

	tracks := make([]model.CanonicalTrack, 0, len(trackTitles))
	for index, trackTitle := range trackTitles {
		tracks = append(tracks, model.CanonicalTrack{
			TrackNumber:     index + 1,
			Title:           trackTitle,
			NormalizedTitle: normalize.Text(trackTitle),
			Artists:         artists,
		})
	}

	return &model.CanonicalAlbum{
		Service:           model.ServiceYouTubeMusic,
		SourceID:          sourceID,
		SourceURL:         canonicalURL,
		Title:             title,
		NormalizedTitle:   normalize.Text(title),
		Artists:           artists,
		NormalizedArtists: normalize.Artists(artists),
		TrackCount:        len(tracks),
		ArtworkURL:        extractFirstGroup(ogImagePattern, body),
		EditionHints:      normalize.EditionHints(title),
		Tracks:            tracks,
	}, nil
}

func youTubeMusicAlbumSourceID(canonicalURL string) string {
	parsed, _ := ParseAlbumURL(canonicalURL)
	if parsed == nil {
		return canonicalURL
	}
	return parsed.ID
}

func extractSearchCandidates(body []byte) []searchCandidate {
	matches := albumResultPattern.FindAllSubmatch(body, -1)
	results := make([]searchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) != 4 {
			continue
		}
		browseID := html.UnescapeString(string(match[2]))
		if browseID == "" {
			continue
		}
		if _, ok := seen[browseID]; ok {
			continue
		}
		seen[browseID] = struct{}{}
		results = append(results, searchCandidate{
			Title:    html.UnescapeString(string(match[1])),
			BrowseID: browseID,
			Artist:   html.UnescapeString(string(match[3])),
		})
	}
	return results
}

func ExtractTrackTitles(body []byte) []string {
	matches := trackTitlePattern.FindAllSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		title := html.UnescapeString(string(match[1]))
		if ShouldSkipTrackTitle(title) {
			continue
		}
		if len(titles) > 0 && titles[len(titles)-1] == title {
			continue
		}
		titles = append(titles, title)
	}
	return titles
}

func ShouldSkipTrackTitle(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	return trackPlayCountPattern.MatchString(strings.ToLower(value))
}

func cleanAlbumTitle(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "\u00a0", " ")
	if index := strings.Index(value, " – "); index > 0 {
		return strings.TrimSpace(value[:index])
	}
	return value
}

func extractFirstGroup(pattern *regexp.Regexp, body []byte) string {
	value, err := htmlx.FirstRegexpGroup(body, pattern, nil)
	if err != nil {
		return ""
	}
	return string(value)
}
