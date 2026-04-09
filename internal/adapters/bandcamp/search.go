package bandcamp

import (
	"html"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var (
	albumSearchResultPattern = regexp.MustCompile(`(?s)<li class="searchresult data-search".*?<div class="itemtype">\s*ALBUM\s*</div>.*?<div class="heading">\s*<a href="([^"]+)">\s*(.*?)\s*</a>.*?(?:<div class="subhead">\s*by\s*(.*?)\s*</div>)?.*?(?:<div class="length">\s*(\d+)\s*tracks,.*?</div>)?.*?(?:<div class="released">\s*released\s*(.*?)\s*</div>)?.*?</li>`)
	songSearchResultPattern  = regexp.MustCompile(`(?s)<li class="searchresult data-search".*?<div class="itemtype">\s*TRACK\s*</div>.*?<div class="heading">\s*<a href="([^"]+)">\s*(.*?)\s*</a>.*?(?:<div class="subhead">\s*by\s*(.*?)\s*</div>)?.*?(?:<div class="released">\s*released\s*(.*?)\s*</div>)?.*?</li>`)
)

type searchCandidate struct {
	URL         string
	Title       string
	Artist      string
	TrackCount  int
	ReleaseDate string
}

type rankedSearchCandidate struct {
	Candidate searchCandidate
	Score     int
}

func extractSearchCandidates(body []byte) []searchCandidate {
	matches := albumSearchResultPattern.FindAllSubmatch(body, -1)
	results := make([]searchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		candidate := searchCandidate{
			URL:         canonicalizeAlbumSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			TrackCount:  parseTrackCount(string(match[4])),
			ReleaseDate: parseReleasedText(string(match[5])),
		}
		if candidate.URL == "" {
			continue
		}
		if _, ok := seen[candidate.URL]; ok {
			continue
		}
		seen[candidate.URL] = struct{}{}
		results = append(results, candidate)
	}
	return results
}

func extractSongSearchCandidates(body []byte) []searchCandidate {
	matches := songSearchResultPattern.FindAllSubmatch(body, -1)
	results := make([]searchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		candidate := searchCandidate{
			URL:         canonicalizeSongSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			ReleaseDate: parseReleasedText(string(match[4])),
		}
		if candidate.URL == "" {
			continue
		}
		if _, ok := seen[candidate.URL]; ok {
			continue
		}
		seen[candidate.URL] = struct{}{}
		results = append(results, candidate)
	}
	return results
}

func rankSearchCandidates(source model.CanonicalAlbum, candidates []searchCandidate) []searchCandidate {
	ranked := make([]rankedSearchCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, rankedSearchCandidate{
			Candidate: candidate,
			Score:     scoreSearchCandidate(source, candidate),
		})
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].Candidate.URL < ranked[j].Candidate.URL
		}
		return ranked[i].Score > ranked[j].Score
	})

	ordered := make([]searchCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		ordered = append(ordered, candidate.Candidate)
	}
	return ordered
}

func scoreSearchCandidate(source model.CanonicalAlbum, candidate searchCandidate) int {
	score := 0

	sourceTitle := normalize.Text(source.Title)
	candidateTitle := normalize.Text(candidate.Title)
	sourceCoreTitle := coreTitle(source.Title)
	candidateCoreTitle := coreTitle(candidate.Title)
	switch {
	case sourceTitle != "" && sourceTitle == candidateTitle:
		score += 40
	case sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle:
		score += 25
	case strings.Contains(candidateTitle, sourceTitle) || strings.Contains(sourceTitle, candidateTitle):
		score += 10
	}

	sourceArtist := ""
	if len(source.Artists) > 0 {
		sourceArtist = normalize.Text(source.Artists[0])
	}
	candidateArtist := normalize.Text(candidate.Artist)
	if sourceArtist != "" && sourceArtist == candidateArtist {
		score += 45
	} else if sourceArtist != "" && strings.Contains(candidateArtist, sourceArtist) {
		score += 20
	}

	if source.TrackCount > 0 && candidate.TrackCount > 0 {
		diff := source.TrackCount - candidate.TrackCount
		if diff < 0 {
			diff = -diff
		}
		switch {
		case diff == 0:
			score += 15
		case diff == 1:
			score += 5
		case diff >= 3:
			score -= 10
		}
	}

	if source.ReleaseDate != "" && candidate.ReleaseDate != "" && len(source.ReleaseDate) >= 4 && len(candidate.ReleaseDate) >= 4 {
		if source.ReleaseDate[:4] == candidate.ReleaseDate[:4] {
			score += 5
		}
	}

	return score
}

func cleanSearchText(value string) string {
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func canonicalizeAlbumSearchURL(value string) string {
	value = html.UnescapeString(value)
	parsed, err := parse.BandcampAlbumURL(value)
	if err != nil {
		return ""
	}
	return parsed.CanonicalURL
}

func canonicalizeSongSearchURL(value string) string {
	value = html.UnescapeString(value)
	parsed, err := parse.BandcampSongURL(value)
	if err != nil {
		return ""
	}
	return parsed.CanonicalURL
}

func parseTrackCount(value string) int {
	count, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return count
}

func parseReleasedText(value string) string {
	value = cleanSearchText(value)
	parts := strings.Fields(value)
	if len(parts) < 3 {
		return ""
	}
	for i := len(parts) - 1; i >= 0; i-- {
		if len(parts[i]) == 4 {
			return parts[i]
		}
	}
	return ""
}

func coreTitle(value string) string {
	normalized := normalize.Text(value)
	for _, marker := range []string{" remastered", " remix", " mix", " deluxe", " super deluxe", " live"} {
		normalized = strings.ReplaceAll(normalized, marker, "")
	}
	return strings.Join(strings.Fields(normalized), " ")
}
