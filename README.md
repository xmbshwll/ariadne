# ariadne

Ariadne is a Go library and CLI for resolving album URLs across music services.

Given one album URL, it fetches canonical metadata from the source service, searches other services for likely equivalents, and ranks matches using identifiers and shared metadata scoring.

## Install

Build the CLI:

```bash
go build -o bin/ariadne ./cmd/ariadne
```

Or run it directly:

```bash
go run ./cmd/ariadne help
```

## CLI

Resolve an album URL:

```bash
go run ./cmd/ariadne resolve https://www.deezer.com/album/12047952
```

Usage:

```bash
ariadne resolve [--apple-music-storefront=us] <album-url>
```

Example output:

```json
{
  "input_url": "https://www.deezer.com/album/12047952",
  "source": {
    "service": "deezer",
    "title": "Abbey Road (Remastered)",
    "artists": ["The Beatles"],
    "release_date": "1969-09-26"
  },
  "links": {
    "spotify": {
      "found": true,
      "summary": "strong",
      "best": {
        "url": "https://open.spotify.com/album/0ETFjACtuP2ADo6LFhL6HN",
        "score": 140,
        "reasons": ["upc exact match", "title exact match"]
      }
    },
    "appleMusic": {
      "found": true,
      "summary": "strong"
    }
  }
}
```

## Library

```go
package main

import (
	"context"
	"fmt"

	"github.com/xmbshwll/ariadne"
)

func main() {
	resolver := ariadne.New(ariadne.LoadConfig())
	resolution, err := resolver.ResolveAlbum(context.Background(), "https://www.deezer.com/album/12047952")
	if err != nil {
		panic(err)
	}
	fmt.Println(resolution.Source.Title)
}
```

You can also build a resolver from custom adapters:

```go
resolver := ariadne.NewWithAdapters(sourceAdapters, targetAdapters)
```

## How it works

1. Parse the input URL.
2. Fetch canonical source metadata.
3. Search target services by UPC, ISRC, and album metadata.
4. Deduplicate and rank candidates.
5. Return the best match and alternates per service.

## Service support

| Service | URL input | Target search | Runtime requirements | Status |
|---|---|---|---|---|
| Spotify | Yes | Yes | Target search requires `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` | supported |
| Apple Music | Yes | Yes | UPC/ISRC search requires `APPLE_MUSIC_KEY_ID`, `APPLE_MUSIC_TEAM_ID`, and `APPLE_MUSIC_PRIVATE_KEY_PATH` | supported |
| Deezer | Yes | Yes | None | supported |
| Bandcamp | Yes | Yes | None; scraping-based | experimental |
| SoundCloud | Yes | Yes | None; public page/API extraction | experimental |
| YouTube Music | Yes | Yes | None; public HTML extraction | experimental |
| TIDAL | Yes | Yes | `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` required | experimental |
| Amazon Music | Parse only | No | Runtime resolution intentionally deferred | deferred |

## Configuration

Environment variables:

- `SPOTIFY_CLIENT_ID`
- `SPOTIFY_CLIENT_SECRET`
- `APPLE_MUSIC_STOREFRONT`
- `APPLE_MUSIC_KEY_ID`
- `APPLE_MUSIC_TEAM_ID`
- `APPLE_MUSIC_PRIVATE_KEY_PATH`
- `TIDAL_CLIENT_ID`
- `TIDAL_CLIENT_SECRET`

See `.env.example` and `docs/configuration.md` for details.

## Documentation

- `docs/configuration.md`
- `docs/service-resolution.md`
- `CHANGELOG.md`
