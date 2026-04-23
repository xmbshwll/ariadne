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

Ariadne is still **pre-v1**.

- public Go API is usable, but may still change before `v1.0.0`
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

Resolve album URL:

```bash
ariadne resolve https://www.deezer.com/album/12047952
```

Resolve song URL:

```bash
ariadne resolve --song https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
```

Let Ariadne auto-detect album vs song:

```bash
ariadne resolve https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
```

Restrict target services:

```bash
ariadne resolve --services=spotify,appleMusic https://www.deezer.com/album/12047952
```

Ask for full details instead of compact links:

```bash
ariadne resolve --verbose https://www.deezer.com/album/12047952
```

By default, CLI prints compact JSON with one best URL per service.

Example:

```json
{
  "deezer": "https://www.deezer.com/album/12047952",
  "spotify": "https://open.spotify.com/album/example",
  "appleMusic": "https://music.apple.com/us/album/example"
}
```

Useful flags:

- `--song` or `--album` to force resource type
- `--verbose` to include metadata, scores, reasons, and alternates
- `--format=json|yaml|csv` to change output format
- `--services=spotify,deezer` to limit target services
- `--min-strength=probable` to hide weaker matches
- `--apple-music-storefront=us` to pick default Apple Music storefront when source URL does not include one
- `--http-timeout=30s` to change per-request timeout
- `--resolution-timeout=30s` to cap whole resolve run
- `--log-level=debug` to print CLI diagnostics to stderr
- `--config=.env` or `--config=path/to/config.yaml` to load config from file

Full command shape:

```bash
ariadne resolve [--log-level=debug] [--song|--album] [--verbose] [--format=json|yaml|csv] [--services=spotify,deezer] [--min-strength=probable] [--apple-music-storefront=us] [--http-timeout=30s] [--resolution-timeout=20s] <url>
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

	albumResolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("album:", albumResolution.Source.Title)
	fmt.Println("spotify:", albumResolution.Matches[ariadne.ServiceSpotify].Best.URL)

	songResolution, err := resolver.ResolveSong(context.Background(), "https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("song:", songResolution.Source.Title)
}
```

Advanced constructors are available when you want more control:

- `ariadne.NewWithClient(...)`
- `ariadne.NewWithAdapters(...)`
- `ariadne.NewWithEntityAdapters(...)`

## How matching works

At a high level, Ariadne does the same thing for every service:

1. parse input URL
2. fetch canonical metadata from source service
3. search each target service
4. deduplicate candidates per service
5. score candidates and return best match plus alternates

When identifiers are available, Ariadne prefers them first:

- album matching prefers `UPC`, then track `ISRC`, then metadata
- song matching prefers `ISRC`, then metadata

That means Spotify, Apple Music, and Deezer usually match more easily than services that rely mostly on fuzzy metadata search.

For detailed runtime behavior by service, see [`docs/service-resolution.md`](./docs/service-resolution.md).

## Service support

| Service | Album input | Album target | Song input | Song target | Notes | Status |
|---|---|---|---|---|---|---|
| Spotify | Yes | Yes | Yes | Yes | Target search needs `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` | supported |
| Apple Music | Yes | Yes | Yes | Yes | UPC and ISRC target search need Apple Music key material | supported |
| Deezer | Yes | Yes | Yes | Yes | No credentials required | supported |
| Bandcamp | Yes | Yes | Yes | Yes | Metadata-first, scraping-based | experimental |
| SoundCloud | Yes | Yes | Yes | Yes | Metadata-first, public page and API extraction | experimental |
| YouTube Music | Yes | Yes | No | No | Public HTML extraction | experimental |
| TIDAL | Yes | Yes | Yes | Yes | Needs `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` | experimental |
| Amazon Music | Parse only | No | No | No | Runtime resolution intentionally deferred | deferred |

## Configuration

Ariadne can read configuration from:

- environment variables through `ariadne.LoadConfig()` in library code
- environment variables in CLI use
- `.env` file or other Viper-supported config file formats in CLI use

Common settings:

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `APPLE_MUSIC_STOREFRONT`
- `APPLE_MUSIC_KEY_ID`
- `APPLE_MUSIC_TEAM_ID`
- `APPLE_MUSIC_PRIVATE_KEY_PATH`
- `TIDAL_CLIENT_ID`
- `TIDAL_CLIENT_SECRET`
- `ARIADNE_HTTP_TIMEOUT`
- `ARIADNE_TARGET_SERVICES`
- `ARIADNE_LOG_LEVEL` for CLI diagnostics

Full configuration guide: [`docs/configuration.md`](./docs/configuration.md)

## Error handling

If you use the library API, branch on exported errors with `errors.Is`, not string matching.

Common exported errors:

- `ariadne.ErrUnsupportedURL`
- `ariadne.ErrNoSourceAdapters`
- `ariadne.ErrResolverNotInitialized`
- `ariadne.ErrAmazonMusicDeferred`
- `ariadne.ErrAppleMusicCredentialsNotConfigured`
- `ariadne.ErrSpotifyCredentialsNotConfigured`
- `ariadne.ErrTIDALCredentialsNotConfigured`
- `ariadne.ErrSourceAdapterReturnedNilParsedURL`
- `ariadne.ErrSourceAdapterReturnedNilAlbum`
- `ariadne.ErrSourceAdapterReturnedNilSong`

Example:

```go
resolution, err := resolver.ResolveAlbum(ctx, inputURL)
if err != nil {
	if errors.Is(err, ariadne.ErrUnsupportedURL) {
		return err
	}
	if errors.Is(err, ariadne.ErrSpotifyCredentialsNotConfigured) {
		return err
	}
	return err
}

_ = resolution
```

## Repository layout

This repository contains two Go modules:

- library: `github.com/xmbshwll/ariadne`
- CLI: `github.com/xmbshwll/ariadne/cmd`

Most users can ignore this. It mainly matters for contributors and releases.

## More docs

- [`docs/configuration.md`](./docs/configuration.md) — config, env vars, and validation tools
- [`docs/service-resolution.md`](./docs/service-resolution.md) — service-by-service runtime behavior
- [`CONTRIBUTING.md`](./CONTRIBUTING.md) — local development and pull request guidance
- [`docs/releasing.md`](./docs/releasing.md) — release steps for both Go modules
- [`CHANGELOG.md`](./CHANGELOG.md) — release history

## License

MIT. See [`LICENSE`](./LICENSE).
