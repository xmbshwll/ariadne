package main

import (
	"context"
	"errors"

	"github.com/xmbshwll/ariadne"
)

var (
	errUnsupportedCLIFixture = errors.New("unsupported")
	errCLIFixtureNotFound    = errors.New("not found")
)

type fixtureSourceAdapterForCLI struct {
	albumByURL map[string]ariadne.CanonicalAlbum
}

func (a fixtureSourceAdapterForCLI) Service() ariadne.ServiceName {
	return "fixture"
}

func (a fixtureSourceAdapterForCLI) ParseAlbumURL(raw string) (*ariadne.ParsedAlbumURL, error) {
	album, ok := a.albumByURL[raw]
	if !ok {
		return nil, errUnsupportedCLIFixture
	}
	return &ariadne.ParsedAlbumURL{Service: album.Service, EntityType: "album", ID: album.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSourceAdapterForCLI) FetchAlbum(_ context.Context, parsed ariadne.ParsedAlbumURL) (*ariadne.CanonicalAlbum, error) {
	album, ok := a.albumByURL[parsed.RawURL]
	if !ok {
		return nil, errCLIFixtureNotFound
	}
	albumCopy := album
	return &albumCopy, nil
}

type fixtureTargetAdapterForCLI struct {
	service     ariadne.ServiceName
	upcResults  []ariadne.CandidateAlbum
	isrcResults []ariadne.CandidateAlbum
	metaResults []ariadne.CandidateAlbum
	metadataErr error
}

func (a fixtureTargetAdapterForCLI) Service() ariadne.ServiceName {
	return a.service
}

func (a fixtureTargetAdapterForCLI) SearchByUPC(_ context.Context, _ string) ([]ariadne.CandidateAlbum, error) {
	return append([]ariadne.CandidateAlbum(nil), a.upcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByISRC(_ context.Context, _ []string) ([]ariadne.CandidateAlbum, error) {
	return append([]ariadne.CandidateAlbum(nil), a.isrcResults...), nil
}

func (a fixtureTargetAdapterForCLI) SearchByMetadata(_ context.Context, _ ariadne.CanonicalAlbum) ([]ariadne.CandidateAlbum, error) {
	if a.metadataErr != nil {
		return nil, a.metadataErr
	}
	return append([]ariadne.CandidateAlbum(nil), a.metaResults...), nil
}

type fixtureSongSourceAdapterForCLI struct {
	songByURL map[string]ariadne.CanonicalSong
}

func (a fixtureSongSourceAdapterForCLI) Service() ariadne.ServiceName {
	return "fixture-song"
}

func (a fixtureSongSourceAdapterForCLI) ParseSongURL(raw string) (*ariadne.ParsedURL, error) {
	song, ok := a.songByURL[raw]
	if !ok {
		return nil, errUnsupportedCLIFixture
	}
	return &ariadne.ParsedURL{Service: song.Service, EntityType: "song", ID: song.SourceID, CanonicalURL: raw, RawURL: raw}, nil
}

func (a fixtureSongSourceAdapterForCLI) FetchSong(_ context.Context, parsed ariadne.ParsedURL) (*ariadne.CanonicalSong, error) {
	song, ok := a.songByURL[parsed.RawURL]
	if !ok {
		return nil, errCLIFixtureNotFound
	}
	songCopy := song
	return &songCopy, nil
}

type fixtureSongTargetAdapterForCLI struct {
	service     ariadne.ServiceName
	isrcResults []ariadne.CandidateSong
	metaResults []ariadne.CandidateSong
	metadataErr error
}

func (a fixtureSongTargetAdapterForCLI) Service() ariadne.ServiceName {
	return a.service
}

func (a fixtureSongTargetAdapterForCLI) SearchSongByISRC(_ context.Context, _ string) ([]ariadne.CandidateSong, error) {
	return append([]ariadne.CandidateSong(nil), a.isrcResults...), nil
}

func (a fixtureSongTargetAdapterForCLI) SearchSongByMetadata(_ context.Context, _ ariadne.CanonicalSong) ([]ariadne.CandidateSong, error) {
	if a.metadataErr != nil {
		return nil, a.metadataErr
	}
	return append([]ariadne.CandidateSong(nil), a.metaResults...), nil
}
