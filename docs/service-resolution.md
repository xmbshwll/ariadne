# How Ariadne resolves albums and songs

This document explains what Ariadne does at runtime for each supported service.

If you want quick version:

- Ariadne parses input URL
- fetches canonical source metadata
- searches each target service
- scores candidates
- returns best match plus alternates

Albums prefer `UPC`, then track `ISRC`, then metadata.
Songs prefer `ISRC`, then metadata.

## Shared rules

### Target services are searched in parallel

Once Ariadne has source metadata, it searches target services concurrently. Each service still runs its own search steps in order.

### Identifier search comes first when available

Identifier-based search is usually stronger than plain metadata search, so Ariadne tries it first.

### Candidate recovery is deliberately tolerant

For core maintained adapters — Spotify, Apple Music, TIDAL, and SoundCloud — Ariadne keeps good candidates it already found even if later search or hydration steps fail.

That means one weak API response does not automatically throw away everything recovered earlier.

## Status labels

| Label | Meaning |
|---|---|
| supported | expected to work well in normal use |
| experimental | works today, but depends on weaker metadata, scraping, unofficial endpoints, or higher-maintenance integrations |
| deferred | URL parsing may exist, but runtime resolution is intentionally not implemented |

## Service summary

| Service | Album input | Album target | Song input | Song target | Search style | Status |
|---|---|---|---|---|---|---|
| Spotify | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Apple Music | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Deezer | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Bandcamp | Yes | Yes | Yes | Yes | metadata only | experimental |
| SoundCloud | Yes | Yes | Yes | Yes | metadata only | experimental |
| YouTube Music | Yes | Yes | No | No | metadata only | experimental |
| TIDAL | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | experimental |
| Amazon Music | Parse only | No | No | No | none | deferred |

## Spotify

### Input support

- album URLs like `https://open.spotify.com/album/{id}` are supported
- song URLs like `https://open.spotify.com/track/{id}` are supported
- if `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` are set, source fetch prefers official Web API
- without credentials, Spotify can still be used as source through public-page bootstrap

### Target search

Spotify target search is enabled only when both Spotify credentials are configured.

Search order:

- albums: `UPC` -> track `ISRC` -> metadata
- songs: `ISRC` -> metadata

### Notes

- Spotify is one of Ariadne's strongest integrations
- canonical output URLs are rebuilt as standard Spotify album or track links

## Apple Music

### Input support

- storefront-aware album URLs such as `https://music.apple.com/us/album/.../{id}` are supported
- storefront-aware song URLs such as `https://music.apple.com/us/album/.../{album-id}?i={track-id}` are supported
- source fetch uses public iTunes Lookup API
- Ariadne keeps storefront from input URL and uses it as region hint

### Target search

Apple Music always supports metadata search through public APIs.

If all of these are set:

- `APPLE_MUSIC_KEY_ID`
- `APPLE_MUSIC_TEAM_ID`
- `APPLE_MUSIC_PRIVATE_KEY_PATH`

Ariadne also enables official MusicKit identifier search.

Search order:

- albums: `UPC` -> track `ISRC` -> metadata
- songs: `ISRC` -> metadata

### Notes

- Apple Music uses public APIs for source fetch and metadata search
- MusicKit is only used for identifier-based search
- storefront matters for lookup and search behavior

## Deezer

### Input support

- album URLs like `https://www.deezer.com/album/{id}` are supported
- song URLs like `https://www.deezer.com/track/{id}` are supported
- source fetch uses public Deezer APIs

### Target search

Search order:

- albums: `UPC` -> track `ISRC` -> metadata
- songs: `ISRC` -> metadata

### Notes

- Deezer is one of Ariadne's strongest low-friction integrations
- no credentials are required
- canonical output URLs are normalized back to standard Deezer links

## Bandcamp

### Input support

- Bandcamp album pages are supported as album input
- Bandcamp track pages are supported as song input
- source fetch uses page HTML and schema.org JSON-LD extraction

### Target search

Bandcamp has no reliable identifier search, so Ariadne relies on metadata search only.

Flow is:

1. search Bandcamp HTML results
2. extract candidate links
3. hydrate candidate pages
4. score hydrated results

### Notes

- Bandcamp can still work well, but it is scraping-based
- metadata quality matters much more here than on identifier-rich services

## SoundCloud

### Input support

- set pages like `https://soundcloud.com/{user}/sets/{slug}` are supported as album-like input
- track pages like `https://soundcloud.com/{user}/{slug}` are supported as song input
- source fetch relies on public page hydration data

### Target search

SoundCloud target search is metadata-first.

Flow is:

1. discover public web `client_id`
2. search public-facing `api-v2`
3. hydrate returned candidates when needed
4. score results

### Notes

- SoundCloud album support is really playlist or set matching
- Ariadne does not treat SoundCloud as reliable identifier-first target
- integration is useful, but still higher-maintenance than Spotify, Apple Music, or Deezer

## YouTube Music

### Input support

- album-like URLs are supported, including `browse` and `playlist?list=` forms
- runtime song input is not implemented yet
- source fetch relies on public HTML extraction

### Target search

YouTube Music album target search is metadata-first:

1. search public page data
2. extract candidate album-like results
3. hydrate candidates
4. score results

### Notes

- album support exists today
- song runtime support is still not implemented
- because this path depends on public HTML behavior, it stays experimental

## TIDAL

### Input support

- album URLs are supported
- song URLs are supported
- source fetch uses official catalog APIs
- both source fetch and target search require `TIDAL_CLIENT_ID` and `TIDAL_CLIENT_SECRET`

### Target search

Search order:

- albums: `UPC` -> `ISRC` -> metadata
- songs: `ISRC` -> metadata

### Notes

- TIDAL matching can be strong when credentials are available
- it stays experimental mostly because credentials are less accessible to typical open-source users

## Amazon Music

### Input support

Amazon Music URLs can be recognized during parsing, but Ariadne does not ship runtime source or target adapters for Amazon Music.

### Target search

None.

### Notes

- current behavior is intentional
- Ariadne returns deferred error instead of pretending runtime support exists
