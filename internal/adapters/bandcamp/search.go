package bandcamp

import (
	"html"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

var (
	albumSearchResultPattern = regexp.MustCompile(`(?s)<li class="searchresult data-search".*?<div class="itemtype">\s*ALBUM\s*</div>.*?<div class="heading">\s*<a href="([^"]+)">\s*(.*?)\s*</a>.*?(?:<div class="subhead">\s*by\s*(.*?)\s*</div>)?.*?(?:<div class="length">\s*(\d+)\s*tracks,.*?</div>)?.*?(?:<div class="released">\s*released\s*(.*?)\s*</div>)?.*?</li>`)
	songSearchResultPattern  = regexp.MustCompile(`(?s)<li class="searchresult data-search".*?<div class="itemtype">\s*TRACK\s*</div>.*?<div class="heading">\s*<a href="([^"]+)">\s*(.*?)\s*</a>.*?(?:<div class="subhead">\s*by\s*(.*?)\s*</div>)?.*?(?:<div class="released">\s*released\s*(.*?)\s*</div>)?.*?</li>`)
)

type SearchCandidate struct {
	URL         string
	Title       string
	Artist      string
	TrackCount  int
	ReleaseDate string
}

type rankedSearchCandidate struct {
	Candidate SearchCandidate
	Score     int
}

func ExtractSearchCandidates(body []byte) []SearchCandidate {
	matches := albumSearchResultPattern.FindAllSubmatch(body, -1)
	return collectSearchCandidates(matches, func(match [][]byte) SearchCandidate {
		return SearchCandidate{
			URL:         canonicalizeAlbumSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			TrackCount:  parseTrackCount(string(match[4])),
			ReleaseDate: parseReleasedText(string(match[5])),
		}
	})
}

func ExtractSongSearchCandidates(body []byte) []SearchCandidate {
	matches := songSearchResultPattern.FindAllSubmatch(body, -1)
	return collectSearchCandidates(matches, func(match [][]byte) SearchCandidate {
		return SearchCandidate{
			URL:         canonicalizeSongSearchURL(string(match[1])),
			Title:       cleanSearchText(string(match[2])),
			Artist:      cleanSearchText(string(match[3])),
			ReleaseDate: parseReleasedText(string(match[4])),
		}
	})
}

func ExtractAutocompleteAlbumSearchCandidates(response fuzzySearchResponse) []SearchCandidate {
	return collectAutocompleteSearchCandidates(response, "a", canonicalizeAlbumSearchURL)
}

func extractAutocompleteSongSearchCandidates(response fuzzySearchResponse) []SearchCandidate {
	return collectAutocompleteSearchCandidates(response, "t", canonicalizeSongSearchURL)
}

func collectAutocompleteSearchCandidates(response fuzzySearchResponse, resultType string, canonicalize func(string) string) []SearchCandidate {
	results := make([]SearchCandidate, 0, len(response.Results))
	seen := make(map[string]struct{}, len(response.Results))
	for _, result := range response.Results {
		if result.Type != resultType {
			continue
		}
		candidate := SearchCandidate{
			URL:    canonicalize(result.URL),
			Title:  cleanSearchText(result.Name),
			Artist: cleanSearchText(result.BandName),
		}
		results = appendUniqueSearchCandidate(results, seen, candidate)
	}
	return results
}

func collectSearchCandidates(matches [][][]byte, build func(match [][]byte) SearchCandidate) []SearchCandidate {
	results := make([]SearchCandidate, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		results = appendUniqueSearchCandidate(results, seen, build(match))
	}
	return results
}

func appendUniqueSearchCandidate(results []SearchCandidate, seen map[string]struct{}, candidate SearchCandidate) []SearchCandidate {
	if candidate.URL == "" {
		return results
	}
	if _, ok := seen[candidate.URL]; ok {
		return results
	}
	seen[candidate.URL] = struct{}{}
	return append(results, candidate)
}

func RankSearchCandidates(source model.CanonicalAlbum, candidates []SearchCandidate) []SearchCandidate {
	return rankCandidates(candidates, func(candidate SearchCandidate) int {
		return scoreSearchCandidate(source, candidate)
	})
}

func rankSongSearchCandidates(source model.CanonicalSong, candidates []SearchCandidate) []SearchCandidate {
	return rankCandidates(candidates, func(candidate SearchCandidate) int {
		return scoreSongSearchCandidate(source, candidate)
	})
}

func rankCandidates(candidates []SearchCandidate, scoreCandidate func(SearchCandidate) int) []SearchCandidate {
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

	ordered := make([]SearchCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		ordered = append(ordered, candidate.Candidate)
	}
	return ordered
}

func scoreSearchCandidate(source model.CanonicalAlbum, candidate SearchCandidate) int {
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

func scoreSongSearchCandidate(source model.CanonicalSong, candidate SearchCandidate) int {
	return scoreSearchMetadata(source.Title, source.Artists, source.ReleaseDate, candidate)
}

func scoreSearchMetadata(sourceTitle string, sourceArtists []string, sourceReleaseDate string, candidate SearchCandidate) int {
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
	value = cleanSearchURL(value)
	parsed, err := ParseAlbumURL(value)
	if err != nil {
		return ""
	}
	return parsed.CanonicalURL
}

func canonicalizeSongSearchURL(value string) string {
	value = cleanSearchURL(value)
	parsed, err := ParseSongURL(value)
	if err != nil {
		return ""
	}
	return parsed.CanonicalURL
}

func cleanSearchURL(value string) string {
	value = html.UnescapeString(strings.TrimSpace(value))
	if index := strings.LastIndex(value, "https://"); index > 0 {
		return value[index:]
	}
	if index := strings.LastIndex(value, "http://"); index > 0 {
		return value[index:]
	}
	return value
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
	for _, v := range slices.Backward(parts) {
		if len(v) == 4 {
			return v
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
