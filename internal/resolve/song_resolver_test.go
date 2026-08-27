package resolve_test

import (
	"context"
	"testing"

	"github.com/xmbshwll/ariadne/internal/adapters"
	resolve "github.com/xmbshwll/ariadne/internal/resolve"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/score"
)

func TestSongResolverResolveSong(t *testing.T) {
	tests := []struct {
		name               string
		resolver           *resolve.SongResolver
		inputURL           string
		wantErr            error
		wantErrMessage     string
		wantSourceService  model.ServiceName
		wantSourceTitle    string
		wantBestCandidates map[model.ServiceName]string
	}{
		{
			name:     "no source adapters",
			resolver: resolve.NewSongs(nil, nil, score.DefaultSongWeights()),
			inputURL: "https://open.spotify.com/track/1",
			wantErr:  resolve.ErrNoSourceAdapters,
		},
		{
			name: "unsupported url",
			resolver: resolve.NewSongs(
				[]adapters.Adapter{newStubSongSourceAdapter()},
				nil,
				score.DefaultSongWeights(),
			),
			inputURL: "https://example.com/track/123",
			wantErr:  resolve.ErrUnsupportedURL,
		},
		{
			name: "collect song candidates and dedupe",
			resolver: resolve.NewSongs(
				[]adapters.Adapter{newStubSongSourceAdapter()},
				[]adapters.Adapter{newStubSongTargetAdapter()},
				score.DefaultSongWeights(),
			),
			inputURL:          "https://open.spotify.com/track/track-1",
			wantSourceService: model.ServiceSpotify,
			wantSourceTitle:   "Come Together",
			wantBestCandidates: map[model.ServiceName]string{
				model.ServiceAppleMusic: "song-1",
			},
		},
		{
			name: "nil source song",
			resolver: resolve.NewSongs(
				[]adapters.Adapter{newNilSongSourceAdapter()},
				nil,
				score.DefaultSongWeights(),
			),
			inputURL:       "https://open.spotify.com/track/track-1",
			wantErrMessage: "fetch source song returned nil from spotify",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolution, err := tt.resolver.ResolveSong(context.Background(), tt.inputURL)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			if tt.wantErrMessage != "" {
				require.Error(t, err)
				assert.Nil(t, resolution)
				assert.EqualError(t, err, tt.wantErrMessage)
				assert.ErrorIs(t, err, resolve.ErrNilSourceSong)
				return
			}
			require.NoError(t, err)

			assert.Equal(t, tt.wantSourceService, resolution.Source.Service)
			assert.Equal(t, tt.wantSourceTitle, resolution.Source.Title)

			for service, wantID := range tt.wantBestCandidates {
				match := resolution.Matches[service]
				require.NotNil(t, match.Best)
				assert.Equal(t, wantID, match.Best.Candidate.CandidateID)
			}
		})
	}
}
