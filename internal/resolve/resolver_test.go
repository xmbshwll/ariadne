package resolve

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

var (
	errUnsupportedTestSource = errors.New("unsupported")
	errTestSourceNotFound    = errors.New("not found")
	errTargetSearchBoom      = errors.New("target search boom")
)

func TestResolverResolveAlbum(t *testing.T) {
	tests := []struct {
		name                string
		resolver            *Resolver
		inputURL            string
		wantErr             error
		wantSourceService   model.ServiceName
		wantSourceTitle     string
		wantTargetServices  []model.ServiceName
		wantCandidateCounts map[model.ServiceName]int
		wantBestCandidates  map[model.ServiceName]string
	}{
		{
			name:     "no source adapters",
			resolver: New(nil, nil, score.DefaultWeights()),
			inputURL: "https://www.deezer.com/album/12047952",
			wantErr:  ErrNoSourceAdapters,
		},
		{
			name: "unsupported url",
			resolver: New(
				[]SourceAdapter{stubSourceAdapter{}},
				nil,
				score.DefaultWeights(),
			),
			inputURL: "https://example.com/album/123",
			wantErr:  ErrUnsupportedURL,
		},
		{
			name: "collect layered candidates and dedupe",
			resolver: New(
				[]SourceAdapter{stubSourceAdapter{}},
				[]TargetAdapter{stubTargetAdapter{}},
				score.DefaultWeights(),
			),
			inputURL:          "https://www.deezer.com/album/12047952",
			wantSourceService: model.ServiceDeezer,
			wantSourceTitle:   "Abbey Road (Remastered)",
			wantTargetServices: []model.ServiceName{
				model.ServiceSpotify,
			},
			wantCandidateCounts: map[model.ServiceName]int{
				model.ServiceSpotify: 2,
			},
			wantBestCandidates: map[model.ServiceName]string{
				model.ServiceSpotify: "album-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := tt.resolver.ResolveAlbum(context.Background(), tt.inputURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantSourceService, resolution.Source.Service)
			assert.Equal(t, tt.wantSourceTitle, resolution.Source.Title)

			for _, service := range tt.wantTargetServices {
				match := resolution.Matches[service]
				candidateCount := len(match.Alternates)
				if match.Best != nil {
					candidateCount++
				}
				assert.Equal(t, tt.wantCandidateCounts[service], candidateCount)
				require.NotNil(t, match.Best)
				assert.Equal(t, tt.wantBestCandidates[service], match.Best.Candidate.CandidateID)
				assert.NotEmpty(t, match.Best.URL)
				assert.NotEmpty(t, match.Best.Reasons)
			}
		})
	}
}

func TestResolverResolveAlbumSearchesTargetsInParallel(t *testing.T) {
	release := make(chan struct{})
	spotifyStarted := make(chan struct{}, 1)
	appleMusicStarted := make(chan struct{}, 1)

	resolver := New(
		[]SourceAdapter{stubSourceAdapter{}},
		[]TargetAdapter{
			blockingTargetAdapter{service: model.ServiceSpotify, started: spotifyStarted, release: release},
			blockingTargetAdapter{service: model.ServiceAppleMusic, started: appleMusicStarted, release: release},
		},
		score.DefaultWeights(),
	)

	resultCh := make(chan error, 1)
	go func() {
		_, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
		resultCh <- err
	}()

	waitStarted := func(name string, started <-chan struct{}) {
		t.Helper()
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			require.FailNowf(t, "target did not start", "timed out waiting for %s target to start", name)
		}
	}

	waitStarted("spotify", spotifyStarted)
	waitStarted("apple music", appleMusicStarted)
	close(release)

	select {
	case err := <-resultCh:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		require.FailNow(t, "timed out waiting for ResolveAlbum to return")
	}
}

type stubSourceAdapter struct{}

func (stubSourceAdapter) Service() model.ServiceName {
	return model.ServiceDeezer
}

func (stubSourceAdapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	if raw != "https://www.deezer.com/album/12047952" {
		return nil, errUnsupportedTestSource
	}
	return &model.ParsedAlbumURL{
		Service:      model.ServiceDeezer,
		EntityType:   "album",
		ID:           "12047952",
		CanonicalURL: raw,
		RawURL:       raw,
	}, nil
}

func (stubSourceAdapter) FetchAlbum(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	return &model.CanonicalAlbum{
		Service:         parsed.Service,
		SourceID:        parsed.ID,
		SourceURL:       parsed.CanonicalURL,
		Title:           "Abbey Road (Remastered)",
		UPC:             "602547670342",
		TrackCount:      17,
		NormalizedTitle: "abbey road remastered",
		Artists:         []string{"The Beatles"},
		Tracks: []model.CanonicalTrack{
			{ISRC: "GBAYE0601690", Title: "Come Together"},
			{ISRC: "GBAYE0601691", Title: "Something"},
		},
	}, nil
}

type stubTargetAdapter struct{}

func (stubTargetAdapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

type blockingTargetAdapter struct {
	service model.ServiceName
	started chan<- struct{}
	release <-chan struct{}
}

type sourceServiceTargetAdapter struct {
	called *bool
}

type failingTargetAdapter struct{}

func (a blockingTargetAdapter) Service() model.ServiceName {
	return a.service
}

func (stubTargetAdapter) SearchByUPC(_ context.Context, upc string) ([]model.CandidateAlbum, error) {
	if upc == "" {
		return nil, nil
	}
	return []model.CandidateAlbum{
		{CandidateID: "album-1", MatchURL: "https://open.spotify.com/album/1", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify, SourceID: "album-1", SourceURL: "https://open.spotify.com/album/1", Title: "Abbey Road (Remastered)", NormalizedTitle: "abbey road remastered", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, UPC: upc, TrackCount: 17, ReleaseDate: "2015-12-24", Tracks: []model.CanonicalTrack{{ISRC: "GBAYE0601690"}, {ISRC: "GBAYE0601691"}}}},
	}, nil
}

func (stubTargetAdapter) SearchByISRC(_ context.Context, isrcs []string) ([]model.CandidateAlbum, error) {
	if len(isrcs) == 0 {
		return nil, nil
	}
	return []model.CandidateAlbum{
		{CandidateID: "album-1", MatchURL: "https://open.spotify.com/album/1", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify, SourceID: "album-1", SourceURL: "https://open.spotify.com/album/1", Title: "Abbey Road (Remastered)", NormalizedTitle: "abbey road remastered", Artists: []string{"The Beatles"}, NormalizedArtists: []string{"the beatles"}, UPC: "602547670342", TrackCount: 17, ReleaseDate: "2015-12-24", Tracks: []model.CanonicalTrack{{ISRC: "GBAYE0601690"}, {ISRC: "GBAYE0601691"}}}},
		{CandidateID: "album-2", MatchURL: "https://open.spotify.com/album/2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify, SourceID: "album-2", SourceURL: "https://open.spotify.com/album/2", Title: "Abbey Road", NormalizedTitle: "abbey road", Artists: []string{"The Beatles Complete On Ukulele"}, NormalizedArtists: []string{"the beatles complete on ukulele"}, TrackCount: 17, ReleaseDate: "2020-01-01", Tracks: []model.CanonicalTrack{{ISRC: "OTHER0001"}}}},
	}, nil
}

func (stubTargetAdapter) SearchByMetadata(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	if album.Title == "" {
		return nil, nil
	}
	return []model.CandidateAlbum{
		{CandidateID: "album-2", MatchURL: "https://open.spotify.com/album/2", CanonicalAlbum: model.CanonicalAlbum{Service: model.ServiceSpotify, SourceID: "album-2", SourceURL: "https://open.spotify.com/album/2", Title: "Abbey Road", NormalizedTitle: "abbey road", Artists: []string{"The Beatles Complete On Ukulele"}, NormalizedArtists: []string{"the beatles complete on ukulele"}, TrackCount: 17, ReleaseDate: "2020-01-01", Tracks: []model.CanonicalTrack{{ISRC: "OTHER0001"}}}},
	}, nil
}

func (a blockingTargetAdapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a blockingTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a blockingTargetAdapter) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	select {
	case a.started <- struct{}{}:
	default:
	}
	<-a.release
	return nil, nil
}

func (a sourceServiceTargetAdapter) Service() model.ServiceName {
	return model.ServiceDeezer
}

func (a sourceServiceTargetAdapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	*a.called = true
	return nil, nil
}

func (a sourceServiceTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	*a.called = true
	return nil, nil
}

func (a sourceServiceTargetAdapter) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	*a.called = true
	return nil, nil
}

func (failingTargetAdapter) Service() model.ServiceName {
	return model.ServiceSpotify
}

func (failingTargetAdapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (failingTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (failingTargetAdapter) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return nil, errTargetSearchBoom
}

func TestResolverCrossServiceFixtures(t *testing.T) {
	fixtures := []struct {
		name            string
		inputURL        string
		source          model.CanonicalAlbum
		targetService   model.ServiceName
		candidates      []model.CandidateAlbum
		wantBestID      string
		wantAlternateID string
	}{
		{
			name:     "prefers remaster over original across deezer to apple music",
			inputURL: "https://fixture.test/deezer/abbey-road-remaster",
			source: testAlbum(model.ServiceDeezer, "src-remaster", "https://fixture.test/deezer/abbey-road-remaster", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{
				releaseDate:     "2015-12-24",
				trackCount:      17,
				totalDurationMS: 2832000,
				editionHints:    []string{"remaster", "remastered"},
				tracks: []model.CanonicalTrack{
					{Title: "Come Together", NormalizedTitle: "come together"},
					{Title: "Something", NormalizedTitle: "something"},
				},
			}),
			targetService: model.ServiceAppleMusic,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceAppleMusic, "apple-remaster", "https://music.apple.com/us/album/remaster/1", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "1969-09-26", trackCount: 17, totalDurationMS: 2831000, editionHints: []string{"remaster", "remastered"}, tracks: []model.CanonicalTrack{{Title: "Come Together", NormalizedTitle: "come together"}, {Title: "Something", NormalizedTitle: "something"}}}),
				testCandidate(model.ServiceAppleMusic, "apple-original", "https://music.apple.com/us/album/original/2", "Abbey Road", []string{"The Beatles"}, testAlbumOptions{releaseDate: "1969-09-26", trackCount: 17, totalDurationMS: 2830000, tracks: []model.CanonicalTrack{{Title: "Come Together", NormalizedTitle: "come together"}, {Title: "Something", NormalizedTitle: "something"}}}),
			},
			wantBestID:      "apple-remaster",
			wantAlternateID: "apple-original",
		},
		{
			name:     "prefers standard over deluxe across spotify to apple music",
			inputURL: "https://fixture.test/spotify/standard-edition",
			source: testAlbum(model.ServiceSpotify, "src-standard", "https://fixture.test/spotify/standard-edition", "Future Nostalgia", []string{"Dua Lipa"}, testAlbumOptions{
				releaseDate:     "2020-03-27",
				trackCount:      11,
				totalDurationMS: 2230000,
				tracks: []model.CanonicalTrack{
					{Title: "Don't Start Now", NormalizedTitle: "dont start now"},
					{Title: "Physical", NormalizedTitle: "physical"},
				},
			}),
			targetService: model.ServiceAppleMusic,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceAppleMusic, "apple-standard", "https://music.apple.com/us/album/standard/3", "Future Nostalgia", []string{"Dua Lipa"}, testAlbumOptions{releaseDate: "2020-03-27", trackCount: 11, totalDurationMS: 2232000, tracks: []model.CanonicalTrack{{Title: "Don't Start Now", NormalizedTitle: "dont start now"}, {Title: "Physical", NormalizedTitle: "physical"}}}),
				testCandidate(model.ServiceAppleMusic, "apple-deluxe", "https://music.apple.com/us/album/deluxe/4", "Future Nostalgia (Moonlight Edition)", []string{"Dua Lipa"}, testAlbumOptions{releaseDate: "2021-02-11", trackCount: 19, totalDurationMS: 3560000, editionHints: []string{"deluxe"}, tracks: []model.CanonicalTrack{{Title: "Don't Start Now", NormalizedTitle: "dont start now"}, {Title: "Physical", NormalizedTitle: "physical"}}}),
			},
			wantBestID:      "apple-standard",
			wantAlternateID: "apple-deluxe",
		},
		{
			name:          "prefers same artist for same title across apple music to spotify",
			inputURL:      "https://fixture.test/apple/same-title",
			source:        testAlbum(model.ServiceAppleMusic, "src-same-title", "https://fixture.test/apple/same-title", "Discovery", []string{"Daft Punk"}, testAlbumOptions{releaseDate: "2001-03-07", trackCount: 14}),
			targetService: model.ServiceSpotify,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSpotify, "spotify-correct", "https://open.spotify.com/album/correct", "Discovery", []string{"Daft Punk"}, testAlbumOptions{releaseDate: "2001-03-07", trackCount: 14}),
				testCandidate(model.ServiceSpotify, "spotify-wrong-artist", "https://open.spotify.com/album/wrong", "Discovery", []string{"Tribute Players"}, testAlbumOptions{releaseDate: "2010-01-01", trackCount: 14}),
			},
			wantBestID:      "spotify-correct",
			wantAlternateID: "spotify-wrong-artist",
		},
		{
			name:          "prefers explicit match over clean across apple music to deezer",
			inputURL:      "https://fixture.test/apple/explicit",
			source:        testAlbum(model.ServiceAppleMusic, "src-explicit", "https://fixture.test/apple/explicit", "DAMN.", []string{"Kendrick Lamar"}, testAlbumOptions{releaseDate: "2017-04-14", trackCount: 14, explicit: true}),
			targetService: model.ServiceDeezer,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceDeezer, "deezer-explicit", "https://www.deezer.com/album/explicit", "DAMN.", []string{"Kendrick Lamar"}, testAlbumOptions{releaseDate: "2017-04-14", trackCount: 14, explicit: true}),
				testCandidate(model.ServiceDeezer, "deezer-clean", "https://www.deezer.com/album/clean", "DAMN.", []string{"Kendrick Lamar"}, testAlbumOptions{releaseDate: "2017-04-14", trackCount: 14, explicit: false}),
			},
			wantBestID:      "deezer-explicit",
			wantAlternateID: "deezer-clean",
		},
		{
			name:     "prefers matching spotify album across tidal source",
			inputURL: "https://fixture.test/tidal/shadows-among-trees",
			source: testAlbum(model.ServiceTIDAL, "src-tidal-shadows", "https://fixture.test/tidal/shadows-among-trees", "Shadows among trees", []string{"Fetch"}, testAlbumOptions{
				releaseDate:     "2020-10-02",
				trackCount:      5,
				totalDurationMS: 2100000,
				tracks: []model.CanonicalTrack{
					{Title: "Kings of mist", NormalizedTitle: "kings of mist"},
					{Title: "Something unspeakable", NormalizedTitle: "something unspeakable"},
				},
			}),
			targetService: model.ServiceSpotify,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSpotify, "spotify-shadows", "https://open.spotify.com/album/shadows", "Shadows among trees", []string{"Fetch"}, testAlbumOptions{releaseDate: "2020-10-02", trackCount: 5, totalDurationMS: 2101000, tracks: []model.CanonicalTrack{{Title: "Kings of mist", NormalizedTitle: "kings of mist"}, {Title: "Something unspeakable", NormalizedTitle: "something unspeakable"}}}),
				testCandidate(model.ServiceSpotify, "spotify-wrong-fetch", "https://open.spotify.com/album/wrong-fetch", "Shadows", []string{"Fetch Tribute"}, testAlbumOptions{releaseDate: "2020-10-02", trackCount: 5}),
			},
			wantBestID:      "spotify-shadows",
			wantAlternateID: "spotify-wrong-fetch",
		},
		{
			name:     "prefers matching tidal album across spotify source",
			inputURL: "https://fixture.test/spotify/brat",
			source: testAlbum(model.ServiceSpotify, "src-spotify-brat", "https://fixture.test/spotify/brat", "BRAT", []string{"Charli XCX"}, testAlbumOptions{
				releaseDate:     "2024-06-07",
				trackCount:      15,
				totalDurationMS: 2500000,
				tracks: []model.CanonicalTrack{
					{Title: "Von dutch", NormalizedTitle: "von dutch"},
					{Title: "360", NormalizedTitle: "360"},
				},
			}),
			targetService: model.ServiceTIDAL,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceTIDAL, "tidal-brat", "https://tidal.com/album/brat", "BRAT", []string{"Charli XCX"}, testAlbumOptions{releaseDate: "2024-06-07", trackCount: 15, totalDurationMS: 2499000, tracks: []model.CanonicalTrack{{Title: "Von dutch", NormalizedTitle: "von dutch"}, {Title: "360", NormalizedTitle: "360"}}}),
				testCandidate(model.ServiceTIDAL, "tidal-brat-remix", "https://tidal.com/album/brat-remix", "BRAT and it's completely different but also still brat", []string{"Charli XCX"}, testAlbumOptions{releaseDate: "2024-10-11", trackCount: 16, totalDurationMS: 2700000, editionHints: []string{"remix"}}),
			},
			wantBestID:      "tidal-brat",
			wantAlternateID: "tidal-brat-remix",
		},
		{
			name:     "prefers standard soundcloud set over deluxe across spotify source",
			inputURL: "https://fixture.test/spotify/cats-dogs",
			source: testAlbum(model.ServiceSpotify, "src-spotify-cats-dogs", "https://fixture.test/spotify/cats-dogs", "Cats & Dogs", []string{"Evidence"}, testAlbumOptions{
				releaseDate:     "2011-09-27",
				trackCount:      17,
				totalDurationMS: 3545000,
				tracks: []model.CanonicalTrack{
					{Title: "The Liner Notes (feat. Aloe Blacc)", NormalizedTitle: "the liner notes feat aloe blacc"},
					{Title: "Strangers", NormalizedTitle: "strangers"},
				},
			}),
			targetService: model.ServiceSoundCloud,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSoundCloud, "evidence-official/sets/cats-dogs-6", "https://soundcloud.com/evidence-official/sets/cats-dogs-6", "Cats & Dogs", []string{"Evidence"}, testAlbumOptions{releaseDate: "2011-09-27", trackCount: 17, totalDurationMS: 3545000, tracks: []model.CanonicalTrack{{Title: "The Liner Notes (feat. Aloe Blacc)", NormalizedTitle: "the liner notes feat aloe blacc"}, {Title: "Strangers", NormalizedTitle: "strangers"}}}),
				testCandidate(model.ServiceSoundCloud, "evidence-official/sets/cats-dogs-3", "https://soundcloud.com/evidence-official/sets/cats-dogs-3", "Cats & Dogs [Deluxe Edition]", []string{"Evidence"}, testAlbumOptions{releaseDate: "2011-09-27", trackCount: 19, totalDurationMS: 3900000, editionHints: []string{"deluxe"}, tracks: []model.CanonicalTrack{{Title: "The Liner Notes (feat. Aloe Blacc)", NormalizedTitle: "the liner notes feat aloe blacc"}, {Title: "Strangers", NormalizedTitle: "strangers"}}}),
			},
			wantBestID:      "evidence-official/sets/cats-dogs-6",
			wantAlternateID: "evidence-official/sets/cats-dogs-3",
		},
		{
			name:     "prefers spotify album over unrelated release across soundcloud source",
			inputURL: "https://fixture.test/soundcloud/cats-dogs",
			source: testAlbum(model.ServiceSoundCloud, "evidence-official/sets/cats-dogs-6", "https://fixture.test/soundcloud/cats-dogs", "Cats & Dogs", []string{"Evidence"}, testAlbumOptions{
				releaseDate:     "2011-09-27",
				trackCount:      17,
				totalDurationMS: 3545000,
				tracks: []model.CanonicalTrack{
					{Title: "The Liner Notes (feat. Aloe Blacc)", NormalizedTitle: "the liner notes feat aloe blacc"},
					{Title: "Strangers", NormalizedTitle: "strangers"},
				},
			}),
			targetService: model.ServiceSpotify,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSpotify, "spotify-cats-dogs", "https://open.spotify.com/album/cats-dogs", "Cats & Dogs", []string{"Evidence"}, testAlbumOptions{releaseDate: "2011-09-27", trackCount: 17, totalDurationMS: 3544000, tracks: []model.CanonicalTrack{{Title: "The Liner Notes (feat. Aloe Blacc)", NormalizedTitle: "the liner notes feat aloe blacc"}, {Title: "Strangers", NormalizedTitle: "strangers"}}}),
				testCandidate(model.ServiceSpotify, "spotify-unrelated-cats", "https://open.spotify.com/album/unrelated-cats", "Cats", []string{"Various Artists"}, testAlbumOptions{releaseDate: "2018-01-01", trackCount: 12}),
			},
			wantBestID:      "spotify-cats-dogs",
			wantAlternateID: "spotify-unrelated-cats",
		},
		{
			name:     "prefers correct artist for same title across deezer to soundcloud",
			inputURL: "https://fixture.test/deezer/discovery",
			source: testAlbum(model.ServiceDeezer, "src-deezer-discovery", "https://fixture.test/deezer/discovery", "Discovery", []string{"Daft Punk"}, testAlbumOptions{
				releaseDate: "2001-03-07",
				trackCount:  14,
				tracks:      []model.CanonicalTrack{{Title: "One More Time", NormalizedTitle: "one more time"}},
			}),
			targetService: model.ServiceSoundCloud,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSoundCloud, "fan-user/sets/discovery", "https://soundcloud.com/fan-user/sets/discovery", "Discovery", []string{"Tribute Players"}, testAlbumOptions{releaseDate: "2010-01-01", trackCount: 14}),
				testCandidate(model.ServiceSoundCloud, "daft-punk/sets/discovery", "https://soundcloud.com/daft-punk/sets/discovery", "Discovery", []string{"Daft Punk"}, testAlbumOptions{releaseDate: "2001-03-07", trackCount: 14, tracks: []model.CanonicalTrack{{Title: "One More Time", NormalizedTitle: "one more time"}}}),
			},
			wantBestID:      "daft-punk/sets/discovery",
			wantAlternateID: "fan-user/sets/discovery",
		},
		{
			name:     "prefers matching spotify album across youtube music source",
			inputURL: "https://fixture.test/youtube-music/abbey-road-super-deluxe",
			source: testAlbum(model.ServiceYouTubeMusic, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", "https://fixture.test/youtube-music/abbey-road-super-deluxe", "Abbey Road (Super Deluxe Edition)", []string{"The Beatles"}, testAlbumOptions{
				releaseDate:     "2019-09-27",
				trackCount:      40,
				totalDurationMS: 7200000,
				editionHints:    []string{"deluxe"},
				tracks: []model.CanonicalTrack{
					{Title: "Come Together (2019 Mix)", NormalizedTitle: "come together 2019 mix"},
					{Title: "Something (2019 Mix)", NormalizedTitle: "something 2019 mix"},
				},
			}),
			targetService: model.ServiceSpotify,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceSpotify, "spotify-abbey-road-super-deluxe", "https://open.spotify.com/album/super-deluxe", "Abbey Road (Super Deluxe Edition)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2019-09-27", trackCount: 40, totalDurationMS: 7199000, editionHints: []string{"deluxe"}, tracks: []model.CanonicalTrack{{Title: "Come Together (2019 Mix)", NormalizedTitle: "come together 2019 mix"}, {Title: "Something (2019 Mix)", NormalizedTitle: "something 2019 mix"}}}),
				testCandidate(model.ServiceSpotify, "spotify-abbey-road-remaster", "https://open.spotify.com/album/remaster", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2009-09-09", trackCount: 17, totalDurationMS: 2832000, editionHints: []string{"remaster"}, tracks: []model.CanonicalTrack{{Title: "Come Together", NormalizedTitle: "come together"}, {Title: "Something", NormalizedTitle: "something"}}}),
			},
			wantBestID:      "spotify-abbey-road-super-deluxe",
			wantAlternateID: "spotify-abbey-road-remaster",
		},
		{
			name:     "prefers matching youtube music deluxe album across spotify source",
			inputURL: "https://fixture.test/spotify/abbey-road-super-deluxe",
			source: testAlbum(model.ServiceSpotify, "src-spotify-abbey-road-super-deluxe", "https://fixture.test/spotify/abbey-road-super-deluxe", "Abbey Road (Super Deluxe Edition)", []string{"The Beatles"}, testAlbumOptions{
				releaseDate:     "2019-09-27",
				trackCount:      40,
				totalDurationMS: 7199000,
				editionHints:    []string{"deluxe"},
				tracks: []model.CanonicalTrack{
					{Title: "Come Together (2019 Mix)", NormalizedTitle: "come together 2019 mix"},
					{Title: "Something (2019 Mix)", NormalizedTitle: "something 2019 mix"},
				},
			}),
			targetService: model.ServiceYouTubeMusic,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceYouTubeMusic, "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", "https://music.youtube.com/playlist?list=OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4", "Abbey Road (Super Deluxe Edition)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2019-09-27", trackCount: 40, totalDurationMS: 7200000, editionHints: []string{"deluxe"}, tracks: []model.CanonicalTrack{{Title: "Come Together (2019 Mix)", NormalizedTitle: "come together 2019 mix"}, {Title: "Something (2019 Mix)", NormalizedTitle: "something 2019 mix"}}}),
				testCandidate(model.ServiceYouTubeMusic, "OLAK5uy_remaster", "https://music.youtube.com/playlist?list=OLAK5uy_remaster", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2009-09-09", trackCount: 17, totalDurationMS: 2832000, editionHints: []string{"remaster"}, tracks: []model.CanonicalTrack{{Title: "Come Together", NormalizedTitle: "come together"}, {Title: "Something", NormalizedTitle: "something"}}}),
			},
			wantBestID:      "OLAK5uy_lqcFZTOPHGwcnP0nYMzNuY0IES0fl7Fe4",
			wantAlternateID: "OLAK5uy_remaster",
		},
		{
			name:     "prefers standard youtube music album over deluxe across deezer source",
			inputURL: "https://fixture.test/deezer/abbey-road-remaster-standard",
			source: testAlbum(model.ServiceDeezer, "src-deezer-abbey-road-standard", "https://fixture.test/deezer/abbey-road-remaster-standard", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{
				releaseDate:     "2009-09-09",
				trackCount:      17,
				totalDurationMS: 2832000,
				editionHints:    []string{"remaster"},
				tracks: []model.CanonicalTrack{
					{Title: "Come Together", NormalizedTitle: "come together"},
					{Title: "Something", NormalizedTitle: "something"},
				},
			}),
			targetService: model.ServiceYouTubeMusic,
			candidates: []model.CandidateAlbum{
				testCandidate(model.ServiceYouTubeMusic, "OLAK5uy_standard", "https://music.youtube.com/playlist?list=OLAK5uy_standard", "Abbey Road (Remastered)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2009-09-09", trackCount: 17, totalDurationMS: 2833000, editionHints: []string{"remaster"}, tracks: []model.CanonicalTrack{{Title: "Come Together", NormalizedTitle: "come together"}, {Title: "Something", NormalizedTitle: "something"}}}),
				testCandidate(model.ServiceYouTubeMusic, "OLAK5uy_super_deluxe", "https://music.youtube.com/playlist?list=OLAK5uy_super_deluxe", "Abbey Road (Super Deluxe Edition)", []string{"The Beatles"}, testAlbumOptions{releaseDate: "2019-09-27", trackCount: 40, totalDurationMS: 7200000, editionHints: []string{"deluxe"}, tracks: []model.CanonicalTrack{{Title: "Come Together (2019 Mix)", NormalizedTitle: "come together 2019 mix"}, {Title: "Something (2019 Mix)", NormalizedTitle: "something 2019 mix"}}}),
			},
			wantBestID:      "OLAK5uy_standard",
			wantAlternateID: "OLAK5uy_super_deluxe",
		},
	}

	sourceAdapter := fixtureSourceAdapter{albumsByURL: make(map[string]model.CanonicalAlbum, len(fixtures))}
	targetCandidates := make(map[model.ServiceName]map[string][]model.CandidateAlbum)
	for _, fixture := range fixtures {
		sourceAdapter.albumsByURL[fixture.inputURL] = fixture.source
		if _, ok := targetCandidates[fixture.targetService]; !ok {
			targetCandidates[fixture.targetService] = make(map[string][]model.CandidateAlbum)
		}
		targetCandidates[fixture.targetService][fixture.source.SourceID] = fixture.candidates
	}

	targets := make([]TargetAdapter, 0, len(targetCandidates))
	for service, candidatesBySourceID := range targetCandidates {
		targets = append(targets, fixtureTargetAdapter{service: service, candidatesBySourceID: candidatesBySourceID})
	}

	resolver := New([]SourceAdapter{sourceAdapter}, targets, score.DefaultWeights())
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			resolution, err := resolver.ResolveAlbum(context.Background(), fixture.inputURL)
			require.NoError(t, err)
			match := resolution.Matches[fixture.targetService]
			require.NotNil(t, match.Best)
			assert.Equal(t, fixture.wantBestID, match.Best.Candidate.CandidateID)
			require.NotEmpty(t, match.Alternates)
			assert.Equal(t, fixture.wantAlternateID, match.Alternates[0].Candidate.CandidateID)
			assert.Greater(t, match.Best.Score, match.Alternates[0].Score)
		})
	}
}

type testAlbumOptions struct {
	releaseDate     string
	trackCount      int
	totalDurationMS int
	editionHints    []string
	explicit        bool
	tracks          []model.CanonicalTrack
}

func testAlbum(service model.ServiceName, sourceID string, sourceURL string, title string, artists []string, opts testAlbumOptions) model.CanonicalAlbum {
	return model.CanonicalAlbum{
		Service:           service,
		SourceID:          sourceID,
		SourceURL:         sourceURL,
		Title:             title,
		NormalizedTitle:   normalizeTitle(title),
		Artists:           append([]string(nil), artists...),
		NormalizedArtists: normalizeArtists(artists),
		ReleaseDate:       opts.releaseDate,
		TrackCount:        opts.trackCount,
		TotalDurationMS:   opts.totalDurationMS,
		EditionHints:      append([]string(nil), opts.editionHints...),
		Explicit:          opts.explicit,
		Tracks:            append([]model.CanonicalTrack(nil), opts.tracks...),
	}
}

func testCandidate(service model.ServiceName, candidateID string, matchURL string, title string, artists []string, opts testAlbumOptions) model.CandidateAlbum {
	album := testAlbum(service, candidateID, matchURL, title, artists, opts)
	return model.CandidateAlbum{CanonicalAlbum: album, CandidateID: candidateID, MatchURL: matchURL}
}

func normalizeTitle(value string) string {
	lower := strings.ToLower(value)
	replacer := strings.NewReplacer("(", " ", ")", " ", ".", " ", "'", "", "!", " ", "-", " ")
	return strings.Join(strings.Fields(replacer.Replace(lower)), " ")
}

func normalizeArtists(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, normalizeTitle(value))
	}
	return out
}

type fixtureSourceAdapter struct {
	albumsByURL map[string]model.CanonicalAlbum
}

func (a fixtureSourceAdapter) Service() model.ServiceName {
	return "fixture"
}

func (a fixtureSourceAdapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	album, ok := a.albumsByURL[raw]
	if !ok {
		return nil, errUnsupportedTestSource
	}
	return &model.ParsedAlbumURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSourceAdapter) FetchAlbum(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	for rawURL, album := range a.albumsByURL {
		if rawURL == parsed.RawURL {
			albumCopy := album
			return &albumCopy, nil
		}
	}
	return nil, errTestSourceNotFound
}

type fixtureTargetAdapter struct {
	service              model.ServiceName
	candidatesBySourceID map[string][]model.CandidateAlbum
}

func (a fixtureTargetAdapter) Service() model.ServiceName {
	return a.service
}

func (a fixtureTargetAdapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a fixtureTargetAdapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a fixtureTargetAdapter) SearchByMetadata(_ context.Context, album model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return append([]model.CandidateAlbum(nil), a.candidatesBySourceID[album.SourceID]...), nil
}
