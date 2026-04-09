package score

import (
	"testing"

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
	if ranking.Best == nil {
		t.Fatalf("expected best candidate")
	}
	if len(ranking.Ranked) != 2 {
		t.Fatalf("ranked count = %d, want 2", len(ranking.Ranked))
	}
	if ranking.Best.Candidate.CandidateID != "best" {
		t.Fatalf("best candidate = %q, want best", ranking.Best.Candidate.CandidateID)
	}
	if ranking.Ranked[0].Score <= ranking.Ranked[1].Score {
		t.Fatalf("expected descending score order, got %d <= %d", ranking.Ranked[0].Score, ranking.Ranked[1].Score)
	}
	if len(ranking.Best.Reasons) == 0 {
		t.Fatalf("expected scoring reasons for best candidate")
	}
}
