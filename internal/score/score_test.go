package score

import (
	"testing"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestRankAlbums(t *testing.T) {
	source := model.CanonicalAlbum{
		Service:           model.ServiceDeezer,
		SourceID:          "12047952",
		SourceURL:         "https://www.deezer.com/album/12047952",
		Title:             "Abbey Road (Remastered)",
		NormalizedTitle:   "abbey road remastered",
		Artists:           []string{"The Beatles"},
		NormalizedArtists: []string{"the beatles"},
		ReleaseDate:       "2015-12-24",
		Label:             "EMI Catalogue",
		UPC:               "602547670342",
		TrackCount:        17,
		TotalDurationMS:   2832000,
		EditionHints:      []string{"remastered"},
		Tracks: []model.CanonicalTrack{
			{ISRC: "GBAYE0601690", Title: "Come Together"},
			{ISRC: "GBAYE0601691", Title: "Something"},
		},
	}

	candidates := []model.CandidateAlbum{
		{
			CandidateID: "best",
			MatchURL:    "https://open.spotify.com/album/best",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceSpotify,
				SourceID:          "best",
				SourceURL:         "https://open.spotify.com/album/best",
				Title:             "Abbey Road (Remastered)",
				NormalizedTitle:   "abbey road remastered",
				Artists:           []string{"The Beatles"},
				NormalizedArtists: []string{"the beatles"},
				ReleaseDate:       "2015-12-24",
				Label:             "EMI Catalogue",
				UPC:               "602547670342",
				TrackCount:        17,
				TotalDurationMS:   2831000,
				EditionHints:      []string{"remastered"},
				Tracks: []model.CanonicalTrack{
					{ISRC: "GBAYE0601690"},
					{ISRC: "GBAYE0601691"},
				},
			},
		},
		{
			CandidateID: "weaker",
			MatchURL:    "https://open.spotify.com/album/weaker",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceSpotify,
				SourceID:          "weaker",
				SourceURL:         "https://open.spotify.com/album/weaker",
				Title:             "Abbey Road",
				NormalizedTitle:   "abbey road",
				Artists:           []string{"The Beatles Complete On Ukulele"},
				NormalizedArtists: []string{"the beatles complete on ukulele"},
				ReleaseDate:       "2020-01-01",
				TrackCount:        17,
				TotalDurationMS:   2700000,
				Tracks: []model.CanonicalTrack{
					{ISRC: "OTHER0001"},
				},
			},
		},
	}

	ranking := RankAlbums(source, candidates)
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

func TestRankAlbumsAppleMusicAlternates(t *testing.T) {
	source := model.CanonicalAlbum{
		Service:           model.ServiceDeezer,
		Title:             "Abbey Road (Remastered)",
		NormalizedTitle:   "abbey road remastered",
		Artists:           []string{"The Beatles"},
		NormalizedArtists: []string{"the beatles"},
		ReleaseDate:       "2015-12-24",
		TrackCount:        17,
		TotalDurationMS:   2832000,
		EditionHints:      []string{"remaster", "remastered"},
	}

	candidates := []model.CandidateAlbum{
		{
			CandidateID: "exact-remaster",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceAppleMusic,
				Title:             "Abbey Road (Remastered)",
				NormalizedTitle:   "abbey road remastered",
				Artists:           []string{"The Beatles"},
				NormalizedArtists: []string{"the beatles"},
				ReleaseDate:       "1969-09-26",
				TrackCount:        17,
				TotalDurationMS:   2831000,
				EditionHints:      []string{"remaster", "remastered"},
			},
		},
		{
			CandidateID: "mix",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceAppleMusic,
				Title:             "Abbey Road (2019 Mix)",
				NormalizedTitle:   "abbey road 2019 mix",
				Artists:           []string{"The Beatles"},
				NormalizedArtists: []string{"the beatles"},
				ReleaseDate:       "1969-09-26",
				TrackCount:        17,
				TotalDurationMS:   2830000,
				EditionHints:      []string{"mix"},
			},
		},
		{
			CandidateID: "super-deluxe",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceAppleMusic,
				Title:             "Abbey Road (Super Deluxe Edition) [2019 Remix & Remaster]",
				NormalizedTitle:   "abbey road super deluxe edition 2019 remix and remaster",
				Artists:           []string{"The Beatles"},
				NormalizedArtists: []string{"the beatles"},
				ReleaseDate:       "1969-09-26",
				TrackCount:        40,
				TotalDurationMS:   6500000,
				EditionHints:      []string{"deluxe", "remaster"},
			},
		},
	}

	ranking := RankAlbums(source, candidates)
	if ranking.Best == nil {
		t.Fatalf("expected best candidate")
	}
	if ranking.Ranked[0].Candidate.CandidateID != "exact-remaster" {
		t.Fatalf("best candidate = %q, want exact-remaster", ranking.Ranked[0].Candidate.CandidateID)
	}
	if ranking.Ranked[1].Candidate.CandidateID != "mix" {
		t.Fatalf("second candidate = %q, want mix", ranking.Ranked[1].Candidate.CandidateID)
	}
	if ranking.Ranked[2].Candidate.CandidateID != "super-deluxe" {
		t.Fatalf("third candidate = %q, want super-deluxe", ranking.Ranked[2].Candidate.CandidateID)
	}
	if ranking.Ranked[1].Score >= ranking.Ranked[0].Score {
		t.Fatalf("mix score = %d, should be lower than exact score %d", ranking.Ranked[1].Score, ranking.Ranked[0].Score)
	}
	if ranking.Ranked[2].Score >= ranking.Ranked[1].Score {
		t.Fatalf("super deluxe score = %d, should be lower than mix score %d", ranking.Ranked[2].Score, ranking.Ranked[1].Score)
	}
}

func TestRankAlbumsPrefersTrackTitleOverlapWithoutIdentifiers(t *testing.T) {
	source := model.CanonicalAlbum{
		Service:           model.ServiceBandcamp,
		Title:             "Live at KEXP",
		NormalizedTitle:   "live at kexp",
		Artists:           []string{"Sea Lemon"},
		NormalizedArtists: []string{"sea lemon"},
		TrackCount:        4,
		Tracks: []model.CanonicalTrack{
			{Title: "Stay", NormalizedTitle: "stay"},
			{Title: "Cellar", NormalizedTitle: "cellar"},
			{Title: "Vaporized", NormalizedTitle: "vaporized"},
			{Title: "Give In", NormalizedTitle: "give in"},
		},
	}

	candidates := []model.CandidateAlbum{
		{
			CandidateID: "high-overlap",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceSpotify,
				Title:             "Live at KEXP",
				NormalizedTitle:   "live at kexp",
				Artists:           []string{"Sea Lemon"},
				NormalizedArtists: []string{"sea lemon"},
				TrackCount:        4,
				Tracks: []model.CanonicalTrack{
					{Title: "Stay", NormalizedTitle: "stay"},
					{Title: "Cellar", NormalizedTitle: "cellar"},
					{Title: "Vaporized", NormalizedTitle: "vaporized"},
					{Title: "Give In", NormalizedTitle: "give in"},
				},
			},
		},
		{
			CandidateID: "low-overlap",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceSpotify,
				Title:             "Live at KEXP",
				NormalizedTitle:   "live at kexp",
				Artists:           []string{"Sea Lemon"},
				NormalizedArtists: []string{"sea lemon"},
				TrackCount:        4,
				Tracks: []model.CanonicalTrack{
					{Title: "Stay", NormalizedTitle: "stay"},
					{Title: "Blue Moon", NormalizedTitle: "blue moon"},
					{Title: "Drive", NormalizedTitle: "drive"},
					{Title: "Night Swim", NormalizedTitle: "night swim"},
				},
			},
		},
	}

	ranking := RankAlbums(source, candidates)
	if ranking.Best == nil {
		t.Fatalf("expected best candidate")
	}
	if ranking.Best.Candidate.CandidateID != "high-overlap" {
		t.Fatalf("best candidate = %q, want high-overlap", ranking.Best.Candidate.CandidateID)
	}
	if ranking.Ranked[0].Score <= ranking.Ranked[1].Score {
		t.Fatalf("expected high-overlap score %d to exceed low-overlap score %d", ranking.Ranked[0].Score, ranking.Ranked[1].Score)
	}
}

func TestRankAlbumsPrefersExplicitVersionOverClean(t *testing.T) {
	source := model.CanonicalAlbum{
		Service:           model.ServiceSpotify,
		Title:             "Midnight City",
		NormalizedTitle:   "midnight city",
		Artists:           []string{"M83"},
		NormalizedArtists: []string{"m83"},
		TrackCount:        10,
		Explicit:          true,
	}

	candidates := []model.CandidateAlbum{
		{
			CandidateID: "explicit",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceAppleMusic,
				Title:             "Midnight City",
				NormalizedTitle:   "midnight city",
				Artists:           []string{"M83"},
				NormalizedArtists: []string{"m83"},
				TrackCount:        10,
				Explicit:          true,
			},
		},
		{
			CandidateID: "clean",
			CanonicalAlbum: model.CanonicalAlbum{
				Service:           model.ServiceAppleMusic,
				Title:             "Midnight City",
				NormalizedTitle:   "midnight city",
				Artists:           []string{"M83"},
				NormalizedArtists: []string{"m83"},
				TrackCount:        10,
				Explicit:          false,
			},
		},
	}

	ranking := RankAlbums(source, candidates)
	if ranking.Best == nil {
		t.Fatalf("expected best candidate")
	}
	if ranking.Best.Candidate.CandidateID != "explicit" {
		t.Fatalf("best candidate = %q, want explicit", ranking.Best.Candidate.CandidateID)
	}
}

func TestRankAlbumsEmpty(t *testing.T) {
	ranking := RankAlbums(model.CanonicalAlbum{}, nil)
	if ranking.Best != nil {
		t.Fatalf("expected nil best candidate")
	}
	if len(ranking.Ranked) != 0 {
		t.Fatalf("ranked count = %d, want 0", len(ranking.Ranked))
	}
}
