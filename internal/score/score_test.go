package score

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	ranking := RankAlbums(source, candidates, DefaultWeights())
	require.NotNil(t, ranking.Best)
	require.Len(t, ranking.Ranked, 2)
	assert.Equal(t, "best", ranking.Best.Candidate.CandidateID)
	assert.Greater(t, ranking.Ranked[0].Score, ranking.Ranked[1].Score)
	assert.NotEmpty(t, ranking.Best.Reasons)
	assert.True(t, ranking.Best.Evidence.Title)
	assert.True(t, ranking.Best.Evidence.Artist)
}

func TestCoreTitleRemovesEditionMarkersOnTokenBoundaries(t *testing.T) {
	assert.Equal(t, "abbey road", coreTitle("Abbey Road (Live)", ""))
	assert.Equal(t, "livewire", coreTitle("Livewire", ""))
}

func TestEditionMarkersPreferLongestMatches(t *testing.T) {
	assert.Equal(t, []string{"super deluxe", "live"}, editionMarkers("Album (Super Deluxe Live)"))
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

	ranking := RankAlbums(source, candidates, DefaultWeights())
	require.NotNil(t, ranking.Best)
	require.Len(t, ranking.Ranked, 3)
	assert.Equal(t, "exact-remaster", ranking.Ranked[0].Candidate.CandidateID)
	assert.Equal(t, "mix", ranking.Ranked[1].Candidate.CandidateID)
	assert.Equal(t, "super-deluxe", ranking.Ranked[2].Candidate.CandidateID)
	assert.Less(t, ranking.Ranked[1].Score, ranking.Ranked[0].Score)
	assert.Less(t, ranking.Ranked[2].Score, ranking.Ranked[1].Score)
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

	ranking := RankAlbums(source, candidates, DefaultWeights())
	require.NotNil(t, ranking.Best)
	require.Len(t, ranking.Ranked, 2)
	assert.Equal(t, "high-overlap", ranking.Best.Candidate.CandidateID)
	assert.Greater(t, ranking.Ranked[0].Score, ranking.Ranked[1].Score)
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

	ranking := RankAlbums(source, candidates, DefaultWeights())
	require.NotNil(t, ranking.Best)
	assert.Equal(t, "explicit", ranking.Best.Candidate.CandidateID)
}

func TestRankAlbumsEmpty(t *testing.T) {
	ranking := RankAlbums(model.CanonicalAlbum{}, nil, DefaultWeights())
	assert.Nil(t, ranking.Best)
	assert.Empty(t, ranking.Ranked)
}
