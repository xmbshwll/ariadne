# Configuration

This guide covers normal Ariadne setup for both library and CLI use.

If you only need quick answer:

- use `ariadne.LoadConfig()` in library code
- use `.env` or environment variables for CLI use
- add Spotify or TIDAL credentials only when you need those official APIs
- add Apple Music key material only when you want MusicKit UPC or ISRC search

## At a glance

| Need | What to set |
|---|---|
| Use library with defaults | nothing; start from `ariadne.LoadConfig()` or `ariadne.DefaultConfig()` |
| Change default Apple Music storefront | `APPLE_MUSIC_STOREFRONT` or `--apple-music-storefront` |
| Change per-request timeout | `ARIADNE_HTTP_TIMEOUT` or `--http-timeout` |
| Limit target services | `ARIADNE_TARGET_SERVICES` or `--services` |
| Enable Spotify target search | `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` |
| Enable Apple Music UPC and ISRC search | `APPLE_MUSIC_KEY_ID`, `APPLE_MUSIC_TEAM_ID`, and `APPLE_MUSIC_PRIVATE_KEY_PATH` |
| Enable TIDAL runtime support | `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` |
| Turn on CLI diagnostics | `ARIADNE_LOG_LEVEL` or `--log-level` |

## Library setup

Library code reads environment variables through `ariadne.LoadConfig()`:

```go
cfg := ariadne.LoadConfig()
resolver := ariadne.New(cfg)
```

You can also build config in code:

```go
cfg := ariadne.DefaultConfig()
cfg.TargetServices = []ariadne.ServiceName{
	ariadne.ServiceSpotify,
	ariadne.ServiceAppleMusic,
}
cfg.HTTPTimeout = 30 * time.Second
resolver := ariadne.New(cfg)
```

A few useful fields:

- `cfg.AppleMusicStorefront` — default Apple Music storefront
- `cfg.HTTPTimeout` — per-request timeout for Ariadne's default HTTP client
- `cfg.TargetServices` — which target services to search
- `cfg.ScoreWeights` — album scoring weights
- `cfg.SongScoreWeights` — song scoring weights

If `cfg.HTTPTimeout` is zero or negative, Ariadne falls back to built-in default.

## CLI setup

CLI can read configuration from:

- command-line flags
- environment variables
- config file from `--config`

Default config file path is `.env`.

CLI precedence is:

1. explicit CLI flags
2. environment variables
3. config file values from `--config`
4. built-in defaults

Examples:

```bash
ariadne resolve https://www.deezer.com/album/12047952
ariadne resolve --song https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
ariadne resolve --services=spotify,appleMusic https://www.deezer.com/album/12047952
ariadne resolve --http-timeout=30s --resolution-timeout=45s https://www.deezer.com/album/12047952
ariadne resolve --config=.env https://www.deezer.com/album/12047952
ariadne resolve --config=./config/ariadne.yaml https://www.deezer.com/album/12047952
ariadne resolve --config="" https://www.deezer.com/album/12047952
```

Use `--config=""` when you want to disable config-file loading entirely.

## Environment variables

### Common runtime settings

| Variable | Default | Used by | What it does |
|---|---|---|---|
| `ARIADNE_HTTP_TIMEOUT` | `15s` | library + CLI | Per-request timeout for Ariadne's default HTTP client. Uses Go duration syntax such as `5s`, `15s`, `30s`, or `1m`. |
| `ARIADNE_TARGET_SERVICES` | unset | library + CLI | Comma-separated target services to search. Example: `spotify,appleMusic,tidal`. When unset, Ariadne uses all enabled default targets. |
| `ARIADNE_LOG_LEVEL` | `error` | CLI only | CLI diagnostics level. Supported values: `error`, `warn`, `info`, `debug`. |

### Spotify

| Variable | Default | What it does |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | empty | Spotify app client ID |
| `SPOTIFY_CLIENT_SECRET` | empty | Spotify app client secret |

When both are set:

- Spotify target search is enabled
- Spotify source fetch prefers official Web API
- Spotify validation command can run against authenticated API

If either value is missing, Spotify can still work as input service through public-page bootstrap, but Spotify target search is disabled.

### Apple Music

| Variable | Default | What it does |
|---|---|---|
| `APPLE_MUSIC_STOREFRONT` | `us` | Default storefront for Apple Music lookups and searches |
| `APPLE_MUSIC_KEY_ID` | empty | Apple Music key ID used for MusicKit developer token generation |
| `APPLE_MUSIC_TEAM_ID` | empty | Apple Developer team ID used for MusicKit developer token generation |
| `APPLE_MUSIC_PRIVATE_KEY_PATH` | empty | Path to Apple `.p8` private key |

Storefront precedence is:

1. `--apple-music-storefront`
2. `APPLE_MUSIC_STOREFRONT`
3. built-in default `us`

Without key material, Ariadne still supports Apple Music source fetch and metadata search.

With all three key-related values set, Ariadne also enables official MusicKit identifier search for:

- album `UPC`
- album-track `ISRC`
- song `ISRC`

### TIDAL

| Variable | Default | What it does |
|---|---|---|
| `TIDAL_CLIENT_ID` | empty | TIDAL client ID |
| `TIDAL_CLIENT_SECRET` | empty | TIDAL client secret |

TIDAL has no public runtime fallback. If credentials are missing, Ariadne cannot use TIDAL official APIs for source fetch or target search.

## Target service names

Use these names in `ARIADNE_TARGET_SERVICES` or `--services`:

- `appleMusic`
- `bandcamp`
- `deezer`
- `soundcloud`
- `spotify`
- `tidal`
- `youtubeMusic`
- `ytmusic` as alias for `youtubeMusic`

`amazonMusic` is not valid target service because runtime resolution is still deferred.

## Local setup

A simple local workflow:

```bash
cp .env.example .env
```

Then fill in what you need, for example:

```bash
SPOTIFY_CLIENT_ID=your-client-id
SPOTIFY_CLIENT_SECRET=your-client-secret
APPLE_MUSIC_STOREFRONT=us
APPLE_MUSIC_KEY_ID=your-apple-key-id
APPLE_MUSIC_TEAM_ID=your-team-id
APPLE_MUSIC_PRIVATE_KEY_PATH=$HOME/keys/AuthKey_XXXXXXXXXX.p8
TIDAL_CLIENT_ID=your-tidal-client-id
TIDAL_CLIENT_SECRET=your-tidal-client-secret
ARIADNE_HTTP_TIMEOUT=30s
ARIADNE_TARGET_SERVICES=spotify,appleMusic
ARIADNE_LOG_LEVEL=error
```

Or export values directly in shell:

```bash
export SPOTIFY_CLIENT_ID=your-client-id
export SPOTIFY_CLIENT_SECRET=your-client-secret
export APPLE_MUSIC_STOREFRONT=us
export APPLE_MUSIC_KEY_ID=your-apple-key-id
export APPLE_MUSIC_TEAM_ID=your-team-id
export APPLE_MUSIC_PRIVATE_KEY_PATH=$HOME/keys/AuthKey_XXXXXXXXXX.p8
export TIDAL_CLIENT_ID=your-tidal-client-id
export TIDAL_CLIENT_SECRET=your-tidal-client-secret
export ARIADNE_HTTP_TIMEOUT=30s
export ARIADNE_TARGET_SERVICES=spotify,appleMusic
export ARIADNE_LOG_LEVEL=error
```

## Debug logging warning

`--log-level=debug` and `ARIADNE_LOG_LEVEL=debug` print effective CLI configuration values, including secrets loaded from environment variables or config files.

Use debug logging carefully.

## Validation commands

These commands are for connector verification and integration debugging. They live in `cmd` module but can be run from repository root through `make`.

### Spotify validation

```bash
make validate-spotify-auth
```

Writes to temporary directory by default. Use `--out-dir <dir>` to keep artifacts.

Artifacts:

- `source-payload-api.json`
- `search-upc-results.json`
- `search-isrc-results.json`
- `search-metadata-results.json`
- `authenticated-summary.json`

### Apple Music validation

```bash
make validate-apple-music-official
```

Writes to temporary directory by default. Use `--out-dir <dir>` to keep artifacts.

Artifacts:

- `source-payload-official.json`
- `search-metadata-official.json`
- `search-upc-official.json` when UPC exists
- `search-isrc-official.json` when track ISRCs exist
- `official-summary.json`

### TIDAL validation

```bash
make validate-tidal-official
```

This command exchanges `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` for bearer token, then writes artifacts to temporary directory by default. Use `--out-dir <dir>` to keep output.

Artifacts:

- `source-payload-official.json`
- `search-albums-official.json`
- `search-upc-official.json`
- `search-isrc-official.json`
- `official-summary.json`

## More detail

For service-by-service behavior, see [`service-resolution.md`](./service-resolution.md).
