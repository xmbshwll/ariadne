# ariadne

[![Go Version](https://img.shields.io/github/go-mod/go-version/xmbshwll/ariadne)](https://go.dev/)
[![CI](https://img.shields.io/github/actions/workflow/status/xmbshwll/ariadne/ci.yml?branch=main)](https://github.com/xmbshwll/ariadne/actions/workflows/ci.yml)
[![License](https://img.shields.io/github/license/xmbshwll/ariadne)](./LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/xmbshwll/ariadne.svg)](https://pkg.go.dev/github.com/xmbshwll/ariadne)
[![Go Report Card](https://goreportcard.com/badge/github.com/xmbshwll/ariadne)](https://goreportcard.com/report/github.com/xmbshwll/ariadne)

Ariadne is a Go library and CLI for turning one music URL into matching album or song links on other services.

Give it a supported Spotify, Apple Music, Deezer, TIDAL, Bandcamp, SoundCloud, or YouTube Music URL. Ariadne fetches source metadata, searches other services, scores candidates, and returns best matches.

## When Ariadne is useful

Use Ariadne when you need to:

- turn one album or song URL into equivalent links on other services
- normalize release and track metadata across providers
- build redirect tools, importers, catalog sync jobs, or internal music tooling
- avoid hand-writing service-specific matching logic

## Current status

Public Go API is stable and guarded by a golden test (`testdata/public_api.txt`); releases are at `v0.7.0` (library) and `cmd/v0.7.0` (CLI), approaching `v1.0.0`.

- Spotify, Apple Music, and Deezer are strongest services today
- Bandcamp, SoundCloud, YouTube Music, and TIDAL are more likely to break or drift
- Amazon Music parsing exists, but runtime resolution is intentionally deferred

## Requirements

- Go `1.26+`

## Install

### Library

```bash
go get github.com/xmbshwll/ariadne
```

### CLI

```bash
go install github.com/xmbshwll/ariadne/cmd/ariadne@latest
```

## Quick start

### CLI

```bash
ariadne resolve https://www.deezer.com/album/12047952
```

Auto-detect album vs song (song URL in, album URL falls through):

```bash
ariadne resolve https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
```

Force the resource type, restrict services, or get full details:

```bash
ariadne resolve --song https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
ariadne resolve --services=spotify,appleMusic https://www.deezer.com/album/12047952
ariadne resolve --verbose https://www.deezer.com/album/12047952
```

By default the CLI prints compact JSON with the best URL per service:

```json
{
  "deezer": "https://www.deezer.com/album/12047952",
  "spotify": "https://open.spotify.com/album/example",
  "appleMusic": "https://music.apple.com/us/album/example"
}
```

Useful flags:

- `--song` / `--album` — force resource type
- `--verbose` — include metadata, scores, reasons, and alternates
- `--format=json|yaml|csv` — output format
- `--services=spotify,deezer` — limit target services
- `--min-strength=probable` — hide weaker matches
- `--apple-music-storefront=us` — default storefront when the URL has none
- `--http-timeout=30s` / `--resolution-timeout=20s` — per-request and whole-run timeouts
- `--log-level=debug` — CLI diagnostics on stderr
- `--config=path/to/file` — load config from `.env`, yaml, json, or toml
- `--version` — print CLI and library versions

Which service is enabled under your config:

```bash
ariadne help resolve
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
	cfg := ariadne.DefaultConfig()
	cfg.TargetServices = []ariadne.ServiceName{
		ariadne.ServiceSpotify,
		ariadne.ServiceAppleMusic,
	}

	resolver := ariadne.New(cfg)

	album, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("album:", album.Source.Title)
	fmt.Println("spotify:", album.Matches[ariadne.ServiceSpotify].Best.URL)
}
```

Construction options are minimal by design: `ariadne.New(config, ariadne.WithHTTPClient(client))` is the only knob. Which services participate is the library's decision — the Provider Catalog picks the adapters for a config, and the public package accepts no adapters of its own.

## How matching works

For every service the pipeline is the same:

1. parse the input URL
2. fetch canonical metadata from the source service
3. search each target service
4. deduplicate candidates per service
5. score candidates and return the best match plus alternates

Identifiers come first, because they are exact:

- albums: `UPC`, then track `ISRC`, then metadata
- songs: `ISRC`, then metadata

That is why Spotify, Apple Music, and Deezer usually match more easily than metadata-only services. A failing target service never fails the whole run — its result carries the error while the others resolve normally.

Per-service runtime behavior: [`docs/service-resolution.md`](./docs/service-resolution.md).

## Service support

| Service | Album input | Album target | Song input | Song target | Notes | Status |
|---|---|---|---|---|---|---|
| Spotify | Yes | Yes | Yes | Yes | Target search needs `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` | supported |
| Apple Music | Yes | Yes | Yes | Yes | UPC and ISRC target search need Apple Music key material | supported |
| Deezer | Yes | Yes | Yes | Yes | No credentials required | supported |
| Bandcamp | Yes | Yes | Yes | Yes | Metadata-first, autocomplete API + JSON-LD hydration | experimental |
| SoundCloud | Yes | Yes | Yes | Yes | Metadata-first, public page and API extraction | experimental |
| YouTube Music | Yes | Yes | Parse only | No | Album public HTML extraction; song hydration deferred | experimental |
| TIDAL | Yes | Yes | Yes | Yes | Needs `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` | experimental |
| Amazon Music | Parse only | No | Parse only | No | Runtime resolution intentionally deferred | deferred |

## Configuration

Configuration comes from:

- environment variables in library code: `ariadne.LoadConfigFromEnv(os.Getenv)`
- environment variables, `.env` files, or Viper-supported config files in CLI use

Common settings: `SPOTIFY_CLIENT_ID`, `SPOTIFY_CLIENT_SECRET`, `APPLE_MUSIC_STOREFRONT`, `APPLE_MUSIC_KEY_ID`, `APPLE_MUSIC_TEAM_ID`, `APPLE_MUSIC_PRIVATE_KEY_PATH`, `TIDAL_CLIENT_ID`, `TIDAL_CLIENT_SECRET`, `ARIADNE_HTTP_TIMEOUT`, `ARIADNE_TARGET_SERVICES`, `ARIADNE_LOG_LEVEL` (CLI).

Full guide: [`docs/configuration.md`](./docs/configuration.md).

## Error handling

Branch on exported errors with `errors.Is`, never string matching:

| Error | When |
|---|---|
| `ariadne.ErrUnsupportedURL` | no source adapter recognized the input URL |
| `ariadne.ErrNoSourceAdapters` | resolver built without source adapters; auto mode treats it as "not a song URL" |
| `ariadne.ErrResolverNotInitialized` | resolve called on a nil or zero Resolver |
| `ariadne.ErrRuntimeDeferred` | URL parses, but hydration is intentionally deferred for that service |
| `ariadne.ErrAmazonMusicDeferred` | Amazon Music variant of the deferred sentinel |
| `ariadne.ErrYouTubeMusicDeferred` | YouTube Music song variant of the deferred sentinel |
| `ariadne.ErrAppleMusicCredentialsNotConfigured` | an Apple Music official API operation needs key material |
| `ariadne.ErrSpotifyCredentialsNotConfigured` | a Spotify Web API operation needs app credentials |
| `ariadne.ErrTIDALCredentialsNotConfigured` | a TIDAL operation needs app credentials |

```go
album, err := resolver.ResolveAlbum(ctx, inputURL)
switch {
case errors.Is(err, ariadne.ErrUnsupportedURL):
	// not a URL any source adapter knows
case errors.Is(err, ariadne.ErrSpotifyCredentialsNotConfigured):
	// set SPOTIFY_CLIENT_ID / SPOTIFY_CLIENT_SECRET
case err != nil:
	// transport or API failure
}
```

The full list is guarded: adding, renaming, or removing an exported name fails a test until reviewed (`testdata/public_api.txt`).

## Repository layout

Two Go modules:

- library: `github.com/xmbshwll/ariadne`
- CLI: `github.com/xmbshwll/ariadne/cmd`

Inside the library, `package ariadne` is the resolve surface — `Config`, `New`, `Resolver`, result types, and errors. Everything else lives under `internal/`:

- `internal/wiring/` — the Provider Catalog: which services act as sources or targets, credential gating, ordering, adapter construction
- `internal/resolve/` — the Entity Resolution pipeline
- `internal/targetsearch/` — Target Search Plans, layer collection, and per-provider candidate collection
- `internal/adapters/` — the eight Music Service adapters (platform code only)
- `internal/canonical/`, `internal/normalize/`, `internal/score/`, `internal/model/` — shared canonical-mapping helpers, text normalization, Scoring, and domain types
- `internal/auth/` — Credential Tokens: the client-credentials token source, Apple Music's signed developer token, and discovered credentials

The CLI reads the Provider Catalog directly through `internal/wiring`; `package ariadne` carries no catalog surface.

## More docs

- [`docs/configuration.md`](./docs/configuration.md) — config, env vars, and validation tools
- [`docs/service-resolution.md`](./docs/service-resolution.md) — service-by-service runtime behavior
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) — local development and pull request guidance
- [`docs/releasing.md`](./docs/releasing.md) — release steps for both Go modules
- [`CHANGELOG.md`](./CHANGELOG.md) — release history

## License

MIT. See [`LICENSE`](./LICENSE).
