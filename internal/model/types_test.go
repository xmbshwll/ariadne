package model_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/xmbshwll/ariadne/internal/model"
)

func TestCandidateSearchKey(t *testing.T) {
	tests := []struct {
		name        string
		service     model.ServiceName
		candidateID string
		matchURL    string
		wantAlbum   string
		wantSong    string
	}{
		{
			name:        "candidate id wins over match url",
			service:     model.ServiceSpotify,
			candidateID: "a1",
			matchURL:    "https://spotify.test/album/a1",
			wantAlbum:   "spotify:id:a1",
			wantSong:    "spotify:id:a1",
		},
		{
			name:      "match url identifies a candidate without a service id",
			service:   model.ServiceDeezer,
			matchURL:  "https://deezer.test/album/1",
			wantAlbum: "deezer:url:https://deezer.test/album/1",
			wantSong:  "deezer:url:https://deezer.test/album/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			album := model.CandidateAlbum{
				CanonicalAlbum: model.CanonicalAlbum{Service: tt.service},
				CandidateID:    tt.candidateID,
				MatchURL:       tt.matchURL,
			}
			song := model.CandidateSong{
				CanonicalSong: model.CanonicalSong{Service: tt.service},
				CandidateID:   tt.candidateID,
				MatchURL:      tt.matchURL,
			}

			assert.Equal(t, tt.wantAlbum, album.SearchKey(), "%s: album SearchKey must key the candidate by service identity", tt.name)
			assert.Equal(t, tt.wantSong, song.SearchKey(), "%s: song SearchKey must key the candidate by service identity", tt.name)
		})
	}
}
