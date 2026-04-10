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
	return collectSearchCandidates(matches, func(match [][]byte) searchCandidate {
		return searchCandidate{
			URL:         canonicalizeAlbumSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			TrackCount:  parseTrackCount(string(match[4])),
			ReleaseDate: parseReleasedText(string(match[5])),
		}
	})
}

func extractSongSearchCandidates(body []byte) []searchCandidate {
	matches := songSearchResultPattern.FindAllSubmatch(body, -1)
	return collectSearchCandidates(matches, func(match [][]byte) searchCandidate {
		return searchCandidate{
			URL:         canonicalizeSongSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			ReleaseDate: parseReleasedText(string(match[4])),
		}
	})
}

func collectSearchCandidates(matches [][][]byte, build func(match [][]byte) searchCandidate) []searchCandidate {
	results := make([]searchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		candidate := build(match)
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
	return rankCandidates(candidates, func(candidate searchCandidate) int {
		return scoreSearchCandidate(source, candidate)
	})
}

func rankSongSearchCandidates(source model.CanonicalSong, candidates []searchCandidate) []searchCandidate {
	return rankCandidates(candidates, func(candidate searchCandidate) int {
		return scoreSongSearchCandidate(source, candidate)
	})
}

func rankCandidates(candidates []searchCandidate, scoreCandidate func(searchCandidate) int) []searchCandidate {
	ranked := make([]rankedSearchCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		ranked = append(ranked, rankedSearchCandidate{
			Candidate: candidate,
			Score:     scoreCandidate(candidate),
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
	score := scoreSearchMetadata(source.Title, source.Artists, source.ReleaseDate, candidate)

	if source.TrackCount <= 0 || candidate.TrackCount <= 0 {
		return score
	}

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

	return score
}

func scoreSongSearchCandidate(source model.CanonicalSong, candidate searchCandidate) int {
	return scoreSearchMetadata(source.Title, source.Artists, source.ReleaseDate, candidate)
}

func scoreSearchMetadata(sourceTitle string, sourceArtists []string, sourceReleaseDate string, candidate searchCandidate) int {
	score := scoreTitle(sourceTitle, candidate.Title)
	score += scoreArtist(sourceArtists, candidate.Artist)
	score += scoreReleaseDate(sourceReleaseDate, candidate.ReleaseDate)
	return score
}

func scoreTitle(sourceTitle string, candidateTitle string) int {
	sourceTitle = normalize.Text(sourceTitle)
	candidateTitle = normalize.Text(candidateTitle)
	sourceCoreTitle := coreTitle(sourceTitle)
	candidateCoreTitle := coreTitle(candidateTitle)
	switch {
	case sourceTitle != "" && sourceTitle == candidateTitle:
		return 40
	case sourceCoreTitle != "" && sourceCoreTitle == candidateCoreTitle:
		return 25
	case strings.Contains(candidateTitle, sourceTitle) || strings.Contains(sourceTitle, candidateTitle):
		return 10
	default:
		return 0
	}
}

func scoreArtist(sourceArtists []string, candidateArtist string) int {
	sourceArtist := ""
	if len(sourceArtists) > 0 {
		sourceArtist = normalize.Text(sourceArtists[0])
	}
	candidateArtist = normalize.Text(candidateArtist)
	switch {
	case sourceArtist != "" && sourceArtist == candidateArtist:
		return 45
	case sourceArtist != "" && strings.Contains(candidateArtist, sourceArtist):
		return 20
	default:
		return 0
	}
}

func scoreReleaseDate(sourceReleaseDate string, candidateReleaseDate string) int {
	if sourceReleaseDate == "" || candidateReleaseDate == "" || len(sourceReleaseDate) < 4 || len(candidateReleaseDate) < 4 {
		return 0
	}
	if sourceReleaseDate[:4] == candidateReleaseDate[:4] {
		return 5
	}
	return 0
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
