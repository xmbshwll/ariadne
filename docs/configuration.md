# Configuration

Ariadne reads runtime configuration from environment variables through `internal/config`.

For the service-by-service runtime resolution path, see `docs/service-resolution.md`.

Current variables:

| Variable | Required | Default | Purpose |
|---|---:|---|---|
| `SPOTIFY_CLIENT_ID` | no | empty | Enables Spotify Web API source/target operations that require app credentials. |
| `SPOTIFY_CLIENT_SECRET` | no | empty | Paired with `SPOTIFY_CLIENT_ID` for Spotify Client Credentials auth. |
| `APPLE_MUSIC_STOREFRONT` | no | `us` | Default storefront used for Apple Music lookup/search when a URL does not already imply one. |
| `APPLE_MUSIC_KEY_ID` | no | empty | Apple Music private key identifier used in the generated MusicKit developer token header. |
| `APPLE_MUSIC_TEAM_ID` | no | empty | Apple Developer team identifier used as the `iss` claim in the generated MusicKit developer token. |
| `APPLE_MUSIC_PRIVATE_KEY_PATH` | no | empty | Path to the downloaded Apple Music `.p8` private key used to sign the generated developer token. |
| `TIDAL_CLIENT_ID` | no | empty | TIDAL client ID used to obtain an OAuth access token for both validation and runtime source/target operations. |
| `TIDAL_CLIENT_SECRET` | no | empty | TIDAL client secret used in the client-credentials token exchange. |

## Credential-gated behavior

This file focuses on credentials and runtime switches. For the full source-fetch and target-search behavior of each connector, see `docs/service-resolution.md`.

### Spotify
- When both `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` are set, Spotify target search is enabled and source fetch prefers the official Web API.
- When either value is missing, Spotify source fetch falls back to the public page bootstrap and Spotify target search is disabled.

### Apple Music
- `APPLE_MUSIC_STOREFRONT` controls the default storefront for Apple Music lookups and metadata search.
- Precedence is:
  1. `--apple-music-storefront=<storefront>`
  2. `APPLE_MUSIC_STOREFRONT`
  3. built-in default: `us`
- Apple Music official validation and identifier search use generated MusicKit tokens from:
  - `APPLE_MUSIC_KEY_ID`
  - `APPLE_MUSIC_TEAM_ID`
  - `APPLE_MUSIC_PRIVATE_KEY_PATH`
- Runtime source fetch and metadata search still use public lookup/search APIs.

### TIDAL
- `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` enable both TIDAL validation and the runtime adapter.
- There is no public TIDAL runtime fallback, so both TIDAL source fetch and TIDAL target search require these variables.

## Setup

Copy the example file and fill in the values you need:

```bash
cp .env.example .env
```

Then export the variables into your shell before running commands, for example:

```bash
export SPOTIFY_CLIENT_ID=your-client-id
export SPOTIFY_CLIENT_SECRET=your-client-secret
export APPLE_MUSIC_STOREFRONT=us
export APPLE_MUSIC_KEY_ID=your-apple-music-key-id
export APPLE_MUSIC_TEAM_ID=your-apple-developer-team-id
export APPLE_MUSIC_PRIVATE_KEY_PATH=$HOME/keys/AuthKey_XXXXXXXXXX.p8
export TIDAL_CLIENT_ID=your-tidal-client-id
export TIDAL_CLIENT_SECRET=your-tidal-client-secret
```

Or source your `.env` file if your shell workflow allows it.

## Validation workflow

To generate authenticated Spotify validation artifacts:

```bash
make validate-spotify-auth
```

This writes:
- `service-samples/spotify/source-payload-api.json`
- `service-samples/spotify/search-upc-results.json`
- `service-samples/spotify/search-isrc-results.json`
- `service-samples/spotify/search-metadata-results.json`
- `service-samples/spotify/authenticated-summary.json`

To generate official Apple Music validation artifacts:

```bash
make validate-apple-music-official
```

This writes:
- `service-samples/apple-music/source-payload-official.json`
- `service-samples/apple-music/search-metadata-official.json`
- `service-samples/apple-music/search-upc-official.json` when UPC is present
- `service-samples/apple-music/search-isrc-official.json` when track ISRCs are present
- `service-samples/apple-music/official-summary.json`

To generate official TIDAL validation artifacts:

```bash
make validate-tidal-official
```

This command first exchanges `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET` for a bearer token, then writes:
- `service-samples/tidal/source-payload-official.json`
- `service-samples/tidal/search-albums-official.json`
- `service-samples/tidal/search-upc-official.json`
- `service-samples/tidal/search-isrc-official.json`
- `service-samples/tidal/official-summary.json`

## Adding new configuration

Add new service credentials in `internal/config/config.go` instead of reading environment variables directly in adapters or commands.

That keeps:
- runtime configuration centralized
- tests simpler
- future service integrations consistent
