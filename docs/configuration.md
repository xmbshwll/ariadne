# Configuration

This guide covers Ariadne setup for library and CLI use.

Quick answer:

- library: start from `ariadne.DefaultConfig()`, or `ariadne.LoadConfigFromEnv(os.Getenv)` to read the environment
- CLI: `.env` file or environment variables, plus flags
- add Spotify or TIDAL credentials only when you need those official APIs
- add Apple Music key material only when you want MusicKit UPC or ISRC search

## At a glance

| Need | What to set |
|---|---|
| Use library with defaults | nothing; start from `ariadne.DefaultConfig()` |
| Change default Apple Music storefront | `APPLE_MUSIC_STOREFRONT` or `--apple-music-storefront` |
| Change per-request timeout | `ARIADNE_HTTP_TIMEOUT` or `--http-timeout` |
| Limit target services | `ARIADNE_TARGET_SERVICES` or `--services` |
| Enable Spotify target search | `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` |
| Enable Apple Music UPC and ISRC search | `APPLE_MUSIC_KEY_ID`, `APPLE_MUSIC_TEAM_ID`, `APPLE_MUSIC_PRIVATE_KEY_PATH` |
| Enable TIDAL runtime support | `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` |
| Turn on CLI diagnostics | `ARIADNE_LOG_LEVEL` or `--log-level` |

## Library setup

Read the environment:

```go
cfg := ariadne.LoadConfigFromEnv(os.Getenv)
resolver := ariadne.New(cfg)
```

Or build config in code:

```go
cfg := ariadne.DefaultConfig()
cfg.TargetServices = []ariadne.ServiceName{
	ariadne.ServiceSpotify,
	ariadne.ServiceAppleMusic,
}
cfg.HTTPTimeout = 30 * time.Second
resolver := ariadne.New(cfg)
```

Useful fields:

- `cfg.AppleMusicStorefront` — default Apple Music storefront
- `cfg.HTTPTimeout` — per-request timeout; zero or negative falls back to the built-in default
- `cfg.TargetServices` — which target services to search

That is the whole public config surface. Ranking weights, adapter selection, and service ordering are Ariadne's decisions and are not caller-tunable.

## CLI setup

The CLI reads configuration from, in precedence order:

1. explicit CLI flags
2. environment variables
3. config file from `--config` (default path `.env`)
4. built-in defaults

```bash
ariadne resolve https://www.deezer.com/album/12047952
ariadne resolve --song https://open.spotify.com/track/2takcwOaAZWiXQijPHIx7B
ariadne resolve --services=spotify,appleMusic https://www.deezer.com/album/12047952
ariadne resolve --http-timeout=30s --resolution-timeout=45s https://www.deezer.com/album/12047952
ariadne resolve --config=./config/ariadne.yaml https://www.deezer.com/album/12047952
ariadne resolve --config="" https://www.deezer.com/album/12047952   # disable file loading
```

Supported config file styles: `.env`-style key=value files, plus Viper-supported formats such as yaml, yml, json, or toml.

## Environment variables

### Common runtime settings

| Variable | Default | Used by | What it does |
|---|---|---|---|
| `ARIADNE_HTTP_TIMEOUT` | `15s` | library + CLI | Per-request timeout for Ariadne's default HTTP client. Go duration syntax: `5s`, `15s`, `30s`, `1m`. |
| `ARIADNE_TARGET_SERVICES` | unset | library + CLI | Comma-separated target services to search, for example `spotify,appleMusic,tidal`. Unset means all enabled default targets. |
| `ARIADNE_LOG_LEVEL` | `error` | CLI only | CLI diagnostics level: `error`, `warn`, `info`, `debug`. |

### Spotify

| Variable | Default | What it does |
|---|---|---|
| `SPOTIFY_CLIENT_ID` | empty | Spotify app client ID |
| `SPOTIFY_CLIENT_SECRET` | empty | Spotify app client secret |

With both set: Spotify target search is enabled and Spotify source fetch prefers the official Web API (public-page bootstrap remains the fallback). With either missing, Spotify still works as an input service through public-page bootstrap, but Spotify target search is disabled.

### Apple Music

| Variable | Default | What it does |
|---|---|---|
| `APPLE_MUSIC_STOREFRONT` | `us` | Default storefront for lookups and searches |
| `APPLE_MUSIC_KEY_ID` | empty | MusicKit developer token key ID |
| `APPLE_MUSIC_TEAM_ID` | empty | Apple Developer team ID |
| `APPLE_MUSIC_PRIVATE_KEY_PATH` | empty | Path to the Apple `.p8` private key |

Storefront precedence: `--apple-music-storefront`, then `APPLE_MUSIC_STOREFRONT`, then built-in `us`.

Without key material, Apple Music source fetch and metadata search still work. With all three key values set, official MusicKit identifier search is also enabled for album `UPC`, album-track `ISRC`, and song `ISRC`.

### TIDAL

| Variable | Default | What it does |
|---|---|---|
| `TIDAL_CLIENT_ID` | empty | TIDAL client ID |
| `TIDAL_CLIENT_SECRET` | empty | TIDAL client secret |

TIDAL has no public runtime fallback. Without credentials, TIDAL official APIs are unavailable for source fetch and target search.

## Target service names

Valid in `ARIADNE_TARGET_SERVICES` and `--services`:

- `appleMusic`
- `bandcamp`
- `deezer`
- `soundcloud`
- `spotify`
- `tidal`
- `youtubeMusic` (alias `ytmusic`)

`amazonMusic` is not a valid target: runtime resolution is still deferred.

## Local setup

```bash
cp .env.example .env
```

Then fill in what you need — see the tables above for the full list. The file is read automatically when you run the CLI from the repository root, or explicitly with `--config=.env`.

## Debug logging warning

`--log-level=debug` and `ARIADNE_LOG_LEVEL=debug` print effective CLI configuration values, including secrets loaded from environment variables or config files. Use carefully.

## Validation commands

For connector verification and integration debugging. They live in the `cmd` module and run from the repository root through `make`. Each tool exchanges credentials, calls its service's API, and writes raw JSON artifacts plus a summary; the shared transport and artifact writing live in `cmd/internal/validation`.

```bash
make validate-spotify-auth        # authenticated Spotify artifacts
make validate-apple-music-official  # MusicKit official artifacts
make validate-tidal-official      # TIDAL official artifacts
```

Each writes to a temporary directory by default; pass `--out-dir <dir>` to keep artifacts.

| Tool | Artifacts |
|---|---|
| Spotify | `source-payload-api.json`, `search-upc-results.json`, `search-isrc-results.json`, `search-metadata-results.json`, `authenticated-summary.json` |
| Apple Music | `source-payload-official.json`, `search-metadata-official.json`, `search-upc-official.json` (when UPC exists), `search-isrc-official.json` (when track ISRCs exist), `official-summary.json` |
| TIDAL | `source-payload-official.json`, `search-albums-official.json`, `search-upc-official.json` (when UPC exists), `search-isrc-official.json` (when track ISRCs exist), `official-summary.json` |

## More detail

For service-by-service behavior, see [`service-resolution.md`](./service-resolution.md).
