package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

func TestRankSongs(t *testing.T) {
	source := model.CanonicalSong{
		Service:                model.ServiceSpotify,
		SourceID:               "track-1",
		SourceURL:              "https://open.spotify.com/track/track-1",
		Title:                  "Come Together",
		NormalizedTitle:        "come together",
		Artists:                []string{"The Beatles"},
		NormalizedArtists:      []string{"the beatles"},
		DurationMS:             259000,
		ISRC:                   "GBAYE0601690",
		TrackNumber:            1,
		AlbumTitle:             "Abbey Road (Remastered)",
		AlbumNormalizedTitle:   "abbey road remastered",
		AlbumArtists:           []string{"The Beatles"},
		AlbumNormalizedArtists: []string{"the beatles"},
		ReleaseDate:            "1969-09-26",
		EditionHints:           []string{"remastered"},
	}

	candidates := []model.CandidateSong{
		{
			CandidateID: "best",
			MatchURL:    "https://music.apple.com/us/song/1",
			CanonicalSong: model.CanonicalSong{
				Service:              model.ServiceAppleMusic,
				SourceID:             "1",
				SourceURL:            "https://music.apple.com/us/song/1",
				Title:                "Come Together",
				NormalizedTitle:      "come together",
				Artists:              []string{"The Beatles"},
				NormalizedArtists:    []string{"the beatles"},
				DurationMS:           258947,
				ISRC:                 "GBAYE0601690",
				TrackNumber:          1,
				AlbumTitle:           "Abbey Road (Remastered)",
				AlbumNormalizedTitle: "abbey road remastered",
				ReleaseDate:          "1969-09-26",
				EditionHints:         []string{"remastered"},
			},
		},
		{
			CandidateID: "weaker",
			MatchURL:    "https://music.apple.com/us/song/2",
			CanonicalSong: model.CanonicalSong{
				Service:              model.ServiceAppleMusic,
				SourceID:             "2",
				SourceURL:            "https://music.apple.com/us/song/2",
				Title:                "Come Together - Live",
				NormalizedTitle:      "come together live",
				Artists:              []string{"Tribute Band"},
				NormalizedArtists:    []string{"tribute band"},
				DurationMS:           310000,
				ISRC:                 "OTHER0001",
				TrackNumber:          7,
				AlbumTitle:           "Abbey Road Live",
				AlbumNormalizedTitle: "abbey road live",
				ReleaseDate:          "2020-01-01",
				EditionHints:         []string{"live"},
			},
		},
	}

	ranking := RankSongs(source, candidates, DefaultSongWeights())
	require.NotNil(t, ranking.Best)
	require.Len(t, ranking.Ranked, 2)
	assert.Equal(t, "best", ranking.Best.Candidate.CandidateID)
	assert.Greater(t, ranking.Ranked[0].Score, ranking.Ranked[1].Score)
	assert.NotEmpty(t, ranking.Best.Reasons)
}
