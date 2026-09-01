// Package canonical holds the shared canonical-mapping helpers every Music
// Service adapter uses to turn wire payloads into canonical album and song
// values. The helpers are deliberately small; what makes the package worth
// existing is that each rule is stated once, so the adapters cannot drift on
// how a release date is truncated, how long an ISO 8601 duration is, or what
// makes a Candidate out of a canonical entity.
package canonical

import (
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/model"
)

// FirstNonEmpty returns the first value that is non-empty after trimming, so
// whitespace-only wire values count as absent. Empty input yields "".
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// SingleArtistList returns a one-artist list for a wire value that carries a
// single free-text artist, or nil when it is blank.
func SingleArtistList(artist string) []string {
	artist = strings.TrimSpace(artist)
	if artist == "" {
		return nil
	}
	return []string{artist}
}

// DateOnly truncates an ISO-like timestamp to its date part, the common shape
// of full timestamps in wire payloads where only the day matters.
func DateOnly(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

// CandidateAlbum wraps a canonical album as that service's search Candidate:
// the SourceID becomes the Candidate ID and the SourceURL the match URL.
func CandidateAlbum(album model.CanonicalAlbum) model.CandidateAlbum {
	return model.CandidateAlbum{
		CanonicalAlbum: album,
		CandidateID:    album.SourceID,
		MatchURL:       album.SourceURL,
	}
}

// CandidateSong wraps a canonical song as that service's search Candidate.
func CandidateSong(song model.CanonicalSong) model.CandidateSong {
	return model.CandidateSong{
		CanonicalSong: song,
		CandidateID:   song.SourceID,
		MatchURL:      song.SourceURL,
	}
}

// ISODurationMilliseconds parses an ISO 8601 duration such as PT1H2M30S or
// PT1.5S into milliseconds. It answers 0 for empty or unparseable input,
// because a wire duration a service did not send is not an error, it is an
// unknown duration.
func ISODurationMilliseconds(value string) int {
	if value == "" {
		return 0
	}
	// ISO 8601 durations are case-insensitive and Go's parser is not.
	lowered := strings.ToLower(value)
	duration, err := time.ParseDuration(strings.TrimPrefix(strings.TrimPrefix(lowered, "p"), "t"))
	if err == nil {
		return int(duration.Milliseconds())
	}

	value = strings.TrimPrefix(value, "P")
	value = strings.TrimPrefix(value, "T")
	var totalSeconds float64
	for len(value) > 0 {
		index := strings.IndexAny(value, "HMS")
		if index <= 0 {
			break
		}
		number := value[:index]
		unit := value[index]
		value = value[index+1:]

		suffix := ""
		switch unit {
		case 'H':
			suffix = "h"
		case 'M':
			suffix = "m"
		case 'S':
			suffix = "s"
		default:
			continue
		}

		parsed, parseErr := time.ParseDuration(number + suffix)
		if parseErr != nil {
			continue
		}
		totalSeconds += parsed.Seconds()
	}
	return int(totalSeconds * 1000)
}
