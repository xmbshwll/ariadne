package resolve

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xmbshwll/ariadne/internal/model"
)

var errEntityResolutionPipelineCollect = errors.New("entity resolution pipeline collect")

type entityResolutionPipelineAdapter struct {
	service model.ServiceName
}

func (a entityResolutionPipelineAdapter) Service() model.ServiceName {
	return a.service
}

func TestResolveEntityPipelineExcludesSourceTargetAndRunsAfterHook(t *testing.T) {
	var collectedMu sync.Mutex
	collectedServices := []model.ServiceName{}
	pipeline := testEntityResolutionPipeline(func(_ context.Context, target entityResolutionPipelineAdapter, _ model.CanonicalAlbum) ([]string, error) {
		collectedMu.Lock()
		collectedServices = append(collectedServices, target.Service())
		collectedMu.Unlock()
		return []string{"candidate"}, nil
	})
	pipeline.after = entityAfterTargetsPolicy[entityResolutionPipelineAdapter, model.CanonicalAlbum, string]{
		errLabel: "resolve after targets",
		run: func(_ context.Context, targets []entityResolutionPipelineAdapter, _ model.CanonicalAlbum, matches map[model.ServiceName]string) error {
			require.Len(t, targets, 1)
			assert.Equal(t, model.ServiceAppleMusic, targets[0].Service())
			matches[model.ServiceBandcamp] = "after-targets"
			return nil
		},
	}

	result, err := resolveEntity(context.Background(), "spotify:album:1", pipeline)

	require.NoError(t, err)
	assert.Equal(t, "spotify:album:1", result.InputURL)
	assert.Equal(t, "1", result.Parsed.ID)
	assert.Equal(t, model.ServiceSpotify, result.Source.Service)
	assert.Equal(t, []model.ServiceName{model.ServiceAppleMusic}, collectedServices)
	assert.NotContains(t, result.Matches, model.ServiceSpotify)
	assert.Equal(t, "appleMusic:candidate", result.Matches[model.ServiceAppleMusic])
	assert.Equal(t, "after-targets", result.Matches[model.ServiceBandcamp])
}

func TestResolveEntityPipelineWrapsTargetErrors(t *testing.T) {
	pipeline := testEntityResolutionPipeline(func(context.Context, entityResolutionPipelineAdapter, model.CanonicalAlbum) ([]string, error) {
		return nil, errEntityResolutionPipelineCollect
	})

	_, err := resolveEntity(context.Background(), "spotify:album:1", pipeline)

	require.Error(t, err)
	assert.ErrorIs(t, err, errEntityResolutionPipelineCollect)
	assert.Contains(t, err.Error(), "resolve target searches: collect candidates from appleMusic")
}

func testEntityResolutionPipeline(
	collect func(context.Context, entityResolutionPipelineAdapter, model.CanonicalAlbum) ([]string, error),
) entityResolutionPipeline[entityResolutionPipelineAdapter, entityResolutionPipelineAdapter, model.ParsedURL, model.CanonicalAlbum, string, []string, string] {
	return entityResolutionPipeline[entityResolutionPipelineAdapter, entityResolutionPipelineAdapter, model.ParsedURL, model.CanonicalAlbum, string, []string, string]{
		sources: []entityResolutionPipelineAdapter{{service: model.ServiceSpotify}},
		targets: []entityResolutionPipelineAdapter{{service: model.ServiceSpotify}, {service: model.ServiceAppleMusic}},
		source: entitySourcePolicy[entityResolutionPipelineAdapter, model.ParsedURL, model.CanonicalAlbum]{
			parse: func(source entityResolutionPipelineAdapter, raw string) (*model.ParsedURL, error) {
				if !strings.HasPrefix(raw, "spotify:") {
					return nil, errUnsupportedTestSource
				}
				return &model.ParsedURL{Service: source.Service(), EntityType: "album", ID: "1"}, nil
			},
			hydrate: func(_ context.Context, source entityResolutionPipelineAdapter, parsed model.ParsedURL) (*model.CanonicalAlbum, error) {
				return &model.CanonicalAlbum{Service: source.Service(), SourceID: parsed.ID, Title: "Source"}, nil
			},
			sourceService: func(source model.CanonicalAlbum) model.ServiceName {
				return source.Service
			},
			entityLabel:  "album",
			nilEntityErr: errNilSourceAlbum,
		},
		target: entityTargetPolicy[entityResolutionPipelineAdapter, model.CanonicalAlbum, string, []string, string]{
			collect: collect,
			rank: func(_ model.CanonicalAlbum, candidates []string) []string {
				return candidates
			},
			result: func(service model.ServiceName, ranking []string) string {
				return string(service) + ":" + strings.Join(ranking, ",")
			},
			candidateLabel: "candidates",
			errLabel:       "resolve target searches",
		},
	}
}
