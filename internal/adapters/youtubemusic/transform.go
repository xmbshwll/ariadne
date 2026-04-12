package youtubemusic

import (
	"errors"
	"html"
	"regexp"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

type searchCandidate struct {
	Title    string
	BrowseID string
	Artist   string
}

func extractAlbum(body []byte, fallbackURL string) (*model.CanonicalAlbum, error) {
	canonicalURL := extractFirstGroup(canonicalURLPattern, body)
	if canonicalURL == "" {
		canonicalURL = strings.TrimSpace(fallbackURL)
	}
	title := cleanAlbumTitle(extractFirstGroup(ogTitlePattern, body))
	if title == "" {
		return nil, errors.Join(errMalformedYouTubeMusicPage, errYouTubeMusicAlbumTitleNotFound)
	}

	artist := html.UnescapeString(extractFirstGroup(subtitleArtistPattern, body))
	trackTitles := extractTrackTitles(body)
	artists := nonEmptyArtistList(artist)
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
	parsed, _ := parse.YouTubeMusicAlbumURL(canonicalURL)
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

func extractTrackTitles(body []byte) []string {
	matches := trackTitlePattern.FindAllSubmatch(body, -1)
	titles := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) != 2 {
			continue
		}
		title := html.UnescapeString(string(match[1]))
		if shouldSkipTrackTitle(title) {
			continue
		}
		if _, ok := seen[title]; ok {
			continue
		}
		seen[title] = struct{}{}
		titles = append(titles, title)
	}
	return titles
}

func shouldSkipTrackTitle(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	lower := strings.ToLower(value)
	return strings.Contains(lower, "wiedergaben") || strings.Contains(lower, "views")
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
	matches := pattern.FindSubmatch(body)
	if len(matches) != 2 {
		return ""
	}
	return html.UnescapeString(string(matches[1]))
}

func nonEmptyArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}

func toCandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{CanonicalAlbum: album, CandidateID: album.SourceID, MatchURL: album.SourceURL}
}
