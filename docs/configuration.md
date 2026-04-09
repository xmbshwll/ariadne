# Configuration

This document explains how to configure Ariadne for normal use and for service validation work.

## Quick start

If you are using the Go library:

```go
cfg := ariadne.LoadConfig()
resolver := ariadne.New(cfg)
```

If you want to limit which services are searched for both album and song resolution:

```go
cfg := ariadne.LoadConfig()
cfg.TargetServices = []ariadne.ServiceName{
	ariadne.ServiceSpotify,
	ariadne.ServiceAppleMusic,
}
resolver := ariadne.New(cfg)
```

If you want to tune album match scoring:

```go
cfg := ariadne.LoadConfig()
cfg.ScoreWeights.TrackTitleStrong = 40
cfg.ScoreWeights.UPCExact = 120
resolver := ariadne.New(cfg)
```

For the service-by-service runtime behavior, including first-wave song support, see [`service-resolution.md`](./service-resolution.md).

## Environment variables

| Variable | Required | Default | What it does |
|---|---:|---|---|
| `SPOTIFY_CLIENT_ID` | no | empty | Enables Spotify Web API operations that need app credentials. |
| `SPOTIFY_CLIENT_SECRET` | no | empty | Used with `SPOTIFY_CLIENT_ID` for Spotify client-credentials auth. |
| `APPLE_MUSIC_STOREFRONT` | no | `us` | Default storefront for Apple Music lookup and metadata search. |
| `APPLE_MUSIC_KEY_ID` | no | empty | Apple Music key ID used to generate a MusicKit developer token. |
| `APPLE_MUSIC_TEAM_ID` | no | empty | Apple Developer team ID used in the MusicKit token. |
| `APPLE_MUSIC_PRIVATE_KEY_PATH` | no | empty | Path to the Apple `.p8` private key used to sign the MusicKit token. |
| `TIDAL_CLIENT_ID` | no | empty | TIDAL client ID used for runtime API access and validation. |
| `TIDAL_CLIENT_SECRET` | no | empty | TIDAL client secret used in the token exchange. |
| `ARIADNE_HTTP_TIMEOUT` | no | `15s` | Per-request timeout for Ariadne's default HTTP client. Uses Go duration syntax such as `5s`, `15s`, `30s`, or `1m`. |

## What changes when credentials are present

### Spotify

- If both `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` are set, Spotify album and song target search are enabled and Spotify source fetch prefers the official Web API.
- If either is missing, Spotify can still be used as an input service through public page bootstrap, but Spotify target search is disabled.

### Apple Music

- `APPLE_MUSIC_STOREFRONT` controls the default storefront for Apple Music album and song lookup/search.
- Storefront precedence is:
  1. CLI flag `--apple-music-storefront=<storefront>`
  2. `APPLE_MUSIC_STOREFRONT`
  3. built-in default: `us`
- If these values are set:
  - `APPLE_MUSIC_KEY_ID`
  - `APPLE_MUSIC_TEAM_ID`
  - `APPLE_MUSIC_PRIVATE_KEY_PATH`
  Ariadne also enables official MusicKit identifier search by album UPC, album-track ISRC, and song ISRC.
- Source fetch and metadata search still use the public lookup/search APIs.

### TIDAL

- `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` are required for the TIDAL runtime adapter.
- There is no public runtime fallback, so both TIDAL album/song source fetch and TIDAL album/song target search require credentials.

## Library vs CLI configuration

### Library

The library reads environment variables through `ariadne.LoadConfig()`.

You can also set `cfg.HTTPTimeout` directly in code to control the default client's per-request timeout.

`cfg.TargetServices` applies to both album and song target selection.

### CLI

The CLI loads configuration with this precedence:

1. explicit CLI flags
2. environment variables
3. config file values from `--config` (defaults to `.env`)
4. built-in defaults

Use `--http-timeout=30s` to override the per-request timeout from the command line.

That means the CLI can work with plain environment variables, a `.env` file, or another config file supported by Viper.

## Local setup

Copy the example file if you want a starting point:

```bash
cp .env.example .env
```

You can then either export variables in your shell:

```bash
export SPOTIFY_CLIENT_ID=your-client-id
export SPOTIFY_CLIENT_SECRET=your-client-secret
export APPLE_MUSIC_STOREFRONT=us
export APPLE_MUSIC_KEY_ID=your-apple-music-key-id
export APPLE_MUSIC_TEAM_ID=your-apple-developer-team-id
export APPLE_MUSIC_PRIVATE_KEY_PATH=$HOME/keys/AuthKey_XXXXXXXXXX.p8
export TIDAL_CLIENT_ID=your-tidal-client-id
export TIDAL_CLIENT_SECRET=your-tidal-client-secret
export ARIADNE_HTTP_TIMEOUT=30s
```

Or let the CLI load `.env` directly, which is the default behavior:

```bash
ariadne resolve https://www.deezer.com/album/12047952
ariadne resolve --song https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
ariadne resolve --config=.env https://www.deezer.com/album/12047952
ariadne resolve --config=./config/ariadne.yaml https://www.deezer.com/album/12047952
```

## Validation tools

The validation commands live in the `cmd` module. From the repository root, use the `make` targets below.

### Spotify

```bash
make validate-spotify-auth
```

By default this writes to a temporary directory and prints the path. Use `--out-dir <dir>` to keep the artifacts in a specific location.

Artifacts written:

- `source-payload-api.json`
- `search-upc-results.json`
- `search-isrc-results.json`
- `search-metadata-results.json`
- `authenticated-summary.json`

### Apple Music

```bash
make validate-apple-music-official
```

By default this writes to a temporary directory and prints the path. Use `--out-dir <dir>` to keep the artifacts in a specific location.

Artifacts written:

- `source-payload-official.json`
- `search-metadata-official.json`
- `search-upc-official.json` when UPC is present
- `search-isrc-official.json` when track ISRCs are present
- `official-summary.json`

### TIDAL

```bash
make validate-tidal-official
```

This first exchanges `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` for a bearer token, then writes to a temporary directory by default. Use `--out-dir <dir>` to keep the artifacts in a specific location.

Artifacts written:

- `source-payload-official.json`
- `search-albums-official.json`
- `search-upc-official.json`
- `search-isrc-official.json`
- `official-summary.json`
