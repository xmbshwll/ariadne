package amazonmusic

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/parse"
)

var ErrDeferredRuntimeAdapter = errors.New("amazon music runtime adapter is deferred: no viable public metadata fetch or search path exists")

type Adapter struct{}

func New(_ *http.Client) *Adapter {
	return &Adapter{}
}

func (a *Adapter) Service() model.ServiceName {
	return model.ServiceAmazonMusic
}

func (a *Adapter) ParseAlbumURL(raw string) (*model.ParsedAlbumURL, error) {
	return parse.AmazonMusicAlbumURL(raw)
}

func (a *Adapter) FetchAlbum(_ context.Context, parsed model.ParsedAlbumURL) (*model.CanonicalAlbum, error) {
	if parsed.Service != model.ServiceAmazonMusic {
		return nil, fmt.Errorf("unexpected service: %s", parsed.Service)
	}
	return nil, ErrDeferredRuntimeAdapter
}

func (a *Adapter) SearchByUPC(_ context.Context, _ string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByISRC(_ context.Context, _ []string) ([]model.CandidateAlbum, error) {
	return nil, nil
}

func (a *Adapter) SearchByMetadata(_ context.Context, _ model.CanonicalAlbum) ([]model.CandidateAlbum, error) {
	return nil, nil
}
