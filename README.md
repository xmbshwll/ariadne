# ariadne

[![Go Version](https://img.shields.io/github/go-mod/go-version/xmbshwll/ariadne)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/xmbshwll/ariadne/ci.yml?branch=main)](https://github.com/xmbshwll/ariadne/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/xmbshwll/ariadne)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/xmbshwll/ariadne.svg)](https://pkg.go.dev/github.com/xmbshwll/ariadne)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmbshwll/ariadne)](https://goreportcard.com/report/github.com/xmbshwll/ariadne)

Ariadne is a Go library and CLI for finding matching album URLs across music services.

Give it one supported album URL and Ariadne will fetch canonical album metadata, search other services for likely matches, and rank the results.

## What it is useful for

Ariadne is a good fit when you need to:

- turn one album URL into equivalent links on other services
- normalize album metadata from different platforms
- build redirect tools, catalog sync jobs, or internal music tooling
- automate cross-service matching without hand-writing service-specific logic

## Current status

Ariadne is currently **pre-v1**.

- The public Go API is usable, but may still change before `v1.0.0`.
- Spotify, Apple Music, and Deezer are the most reliable services today.
- Bandcamp, SoundCloud, YouTube Music, and TIDAL are more volatile.
- Amazon Music URL parsing exists, but runtime resolution is intentionally deferred.

## Install

### Library

```bash
go get github.com/xmbshwll/ariadne
```

### CLI

```bash
go install github.com/xmbshwll/ariadne/cmd/ariadne@latest
```

Ariadne currently requires **Go 1.26+**.

## Quick start

### CLI

Resolve an album URL:

```bash
ariadne resolve https://www.deezer.com/album/12047952
```

By default, Ariadne prints compact JSON with the best link it found for each service.

Example:

```json
{
  "deezer": "https://www.deezer.com/album/12047952",
  "spotify": "https://open.spotify.com/album/example",
  "appleMusic": "https://music.apple.com/us/album/example"
}
```

Useful flags:

- `--verbose` to include source metadata, scores, reasoning, and alternates
- `--format=json|yaml|csv` to change output format
- `--services=spotify,deezer` to limit which target services are searched
- `--min-strength=probable` to hide weaker matches
- `--http-timeout=30s` to raise or lower the per-request HTTP timeout
- `--config=.env` or `--config=path/to/config.yaml` to load config from a file

Full command shape:

```bash
ariadne resolve [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--http-timeout=30s] <album-url>
```

### Library

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/xmbshwll/ariadne"
)

func main() {
	cfg := ariadne.LoadConfig()
	cfg.TargetServices = []ariadne.ServiceName{
		ariadne.ServiceSpotify,
		ariadne.ServiceAppleMusic,
	}

	resolver := ariadne.New(cfg)
	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("source:", resolution.Source.Title)
	fmt.Println("spotify:", resolution.Matches[ariadne.ServiceSpotify].Best.URL)
}
```

If you want to supply your own adapters, you can also build a resolver with:

```go
resolver := ariadne.NewWithAdapters(sourceAdapters, targetAdapters)
```

## How matching works

Ariadne uses the same matching pipeline for every supported service.

### 1. Parse and canonicalize the source album

The input URL is first parsed by the registered source adapters until one recognizes it.
That adapter then fetches the source album and converts it into a shared canonical shape with fields such as:

- album title
- credited artists
- release date
- label
- UPC
- track list
- track ISRCs
- total duration
- explicit flag
- edition hints such as remaster or deluxe

This gives the resolver one normalized source record regardless of which service the input came from.

### 2. Search each target service in layers

Each target service is searched independently, and the source service is skipped.
For each target, Ariadne collects candidates in this order:

1. **UPC search** when the source album has a UPC
2. **ISRC search** when the source tracks expose ISRCs
3. **Metadata search** using album title and artist queries

Metadata search is the fallback that keeps the resolver useful for services that expose weak identifiers or none at all. It uses search-oriented variants of the source metadata, including split artist credits and alternate title forms when available.

### 3. Deduplicate candidates

Results collected from UPC, ISRC, and metadata search are merged and deduplicated per service. If the same album is found through multiple paths, it is scored only once.

### 4. Score every candidate with shared signals

Ariadne then ranks all candidates for a target service with a shared scoring model. The score combines positive and negative signals, including:

- exact UPC match
- strong or partial ISRC overlap
- exact title match
- core title match after removing edition markers
- exact primary artist match or broader artist overlap
- strong or partial track-title overlap
- exact or near track-count match
- exact release-date match or same-year match
- near total duration
- exact label match
- penalties for explicit mismatches
- penalties for edition mismatches such as remaster vs non-remaster

This matters because no single signal is reliable across every service. A candidate can still rank well even when, for example, UPC is missing, as long as the artist, track list, date, and title line up.

### 5. Sort and return best match plus alternates

Candidates are sorted by descending score. For each target service, Ariadne returns:

- the best candidate
- lower-ranked alternates
- the score and human-readable reasons when `--verbose` is enabled

### Confidence bands

Raw scores are also mapped into user-facing confidence bands:

- `strong`: `>= 100`
- `probable`: `>= 70`
- `weak`: `>= 50`
- `very_weak`: `< 50`

The CLI uses these bands for `--min-strength` filtering.

### Practical consequence

Identifier-rich sources such as Spotify, Apple Music, or Deezer usually match more easily because UPC and ISRC search can fire early.
Sources such as Bandcamp often rely much more heavily on metadata search, so title normalization, alternate titles, and track-level overlap matter more there.

## Service support

| Service | Can use as input | Can search as target | Requirements | Status |
|---|---|---|---|---|
| Spotify | Yes | Yes | Target search requires `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` | supported |
| Apple Music | Yes | Yes | UPC/ISRC search requires `APPLE_MUSIC_KEY_ID`, `APPLE_MUSIC_TEAM_ID`, and `APPLE_MUSIC_PRIVATE_KEY_PATH` | supported |
| Deezer | Yes | Yes | None | supported |
| Bandcamp | Yes | Yes | None; scraping-based | experimental |
| SoundCloud | Yes | Yes | None; public page/API extraction | experimental |
| YouTube Music | Yes | Yes | None; public HTML extraction | experimental |
| TIDAL | Yes | Yes | `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` | experimental |
| Amazon Music | Parse only | No | Runtime resolution intentionally deferred | deferred |

For a detailed explanation of how each service works at runtime, see [`docs/service-resolution.md`](./docs/service-resolution.md).

## Configuration

### Library configuration

The library reads environment variables through `ariadne.LoadConfig()`.

Supported environment variables:

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `APPLE_MUSIC_STOREFRONT`
- `APPLE_MUSIC_KEY_ID`
- `APPLE_MUSIC_TEAM_ID`
- `APPLE_MUSIC_PRIVATE_KEY_PATH`
- `TIDAL_CLIENT_ID`
- `TIDAL_CLIENT_SECRET`
- `ARIADNE_HTTP_TIMEOUT` — per-request HTTP timeout as a Go duration such as `15s`, `30s`, or `1m`

Ranking weights are configured in code through `ariadne.Config.ScoreWeights`.

### CLI configuration

The CLI loads configuration with this precedence:

1. explicit CLI flags
2. environment variables
3. a config file passed with `--config` (defaults to `.env`)
4. built-in defaults

CLI output filtering is controlled with flags such as `--services` and `--min-strength`. The CLI also accepts `--http-timeout` to override the default per-request timeout.

For more detail, see [`docs/configuration.md`](./docs/configuration.md).

## Error handling

If you are using the Go library, branch on resolver failures with `errors.Is`, not string matching.

Public sentinel errors:

- `ariadne.ErrUnsupportedURL`
  - no registered source adapter recognized the input URL
- `ariadne.ErrNoSourceAdapters`
  - the resolver was built without any source adapters
- `ariadne.ErrAmazonMusicDeferred`
  - an Amazon Music URL was recognized, but runtime resolution is intentionally deferred
- `ariadne.ErrAppleMusicCredentialsNotConfigured`
  - an Apple Music official API operation needs developer token credentials
- `ariadne.ErrSpotifyCredentialsNotConfigured`
  - a Spotify Web API operation needs app credentials
- `ariadne.ErrTIDALCredentialsNotConfigured`
  - a TIDAL source or target operation needs app credentials

Example:

```go
resolution, err := resolver.ResolveAlbum(ctx, inputURL)
if err != nil {
	if errors.Is(err, ariadne.ErrUnsupportedURL) {
		return err
	}
	if errors.Is(err, ariadne.ErrAppleMusicCredentialsNotConfigured) {
		return err
	}
	if errors.Is(err, ariadne.ErrSpotifyCredentialsNotConfigured) {
		return err
	}
	if errors.Is(err, ariadne.ErrTIDALCredentialsNotConfigured) {
		return err
	}
	return err
}

_ = resolution
```

## Modules and versioning

This repository contains two Go modules:

- library: `github.com/xmbshwll/ariadne`
- CLI: `github.com/xmbshwll/ariadne/cmd`

Version tags follow normal Go submodule conventions:

- library tags: `vX.Y.Z`
- CLI tags: `cmd/vX.Y.Z`

Most users can ignore this unless they are packaging, contributing, or cutting releases.

## Development

Common commands:

```bash
make build
make test
make test-race
make lint
make verify
make verify-release
```

- [`CONTRIBUTING.md`](./CONTRIBUTING.md) explains local development and contribution flow.
- [`docs/releasing.md`](./docs/releasing.md) documents the release process.
- `cmd/validate-spotify-auth`, `cmd/validate-apple-music-official`, and `cmd/validate-tidal-official` are maintainer utilities for checking service integrations. They write to a temporary directory by default unless `--out-dir` is provided.

## Documentation

- [`docs/configuration.md`](./docs/configuration.md)
- [`docs/service-resolution.md`](./docs/service-resolution.md)
- [`CHANGELOG.md`](./CHANGELOG.md)
- [`CONTRIBUTING.md`](./CONTRIBUTING.md)
- [`docs/releasing.md`](./docs/releasing.md)

## License

MIT. See [`LICENSE`](./LICENSE).
