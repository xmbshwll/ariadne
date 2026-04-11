package main

import "github.com/xmbshwll/ariadne"

func newCLIResolution(resolution ariadne.Resolution) cliResolution {
	links := make(map[string]cliMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		links[string(service)] = newCLIMatchResult(match)
	}

	return cliResolution{
		InputURL: resolution.InputURL,
		Source:   newCLIAlbum(resolution.Source),
		Links:    links,
	}
}

func newCLILinks(resolution ariadne.Resolution) map[string]string {
	links := map[string]string{}
	if resolution.Source.Service != "" && resolution.Source.SourceURL != "" {
		links[string(resolution.Source.Service)] = resolution.Source.SourceURL
	}
	for service, match := range resolution.Matches {
		if match.Best == nil || match.Best.URL == "" {
			continue
		}
		if _, exists := links[string(service)]; exists {
			continue
		}
		links[string(service)] = match.Best.URL
	}
	return links
}

func newCLISongResolution(resolution ariadne.SongResolution) cliSongResolution {
	links := make(map[string]cliSongMatchResult, len(resolution.Matches))
	for service, match := range resolution.Matches {
		links[string(service)] = newCLISongMatchResult(match)
	}

	return cliSongResolution{
		InputURL: resolution.InputURL,
		Source:   newCLISong(resolution.Source),
		Links:    links,
	}
}

func newCLISongLinks(resolution ariadne.SongResolution) map[string]string {
	links := map[string]string{}
	if resolution.Source.Service != "" && resolution.Source.SourceURL != "" {
		links[string(resolution.Source.Service)] = resolution.Source.SourceURL
	}
	for service, match := range resolution.Matches {
		if match.Best == nil || match.Best.URL == "" {
			continue
		}
		if _, exists := links[string(service)]; exists {
			continue
		}
		links[string(service)] = match.Best.URL
	}
	return links
}
