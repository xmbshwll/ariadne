# How Ariadne resolves albums and songs

This document explains what Ariadne actually does at runtime for each supported service.

## The shared resolution pipeline

For every supported input URL, Ariadne follows the same high-level flow:

1. parse the input URL as an album or song
2. fetch canonical metadata from the source service
3. search each target service in identifier-first layers, where supported
4. deduplicate candidates
5. score them with the entity-appropriate ranking logic
6. return the best match and alternates per service

Album resolution prefers `UPC`, then track `ISRC`, then album metadata.
Song resolution prefers song `ISRC`, then song metadata.

Not every service supports every search step. The sections below describe the current runtime path for each service.

## Resolver concurrency and failure model

Ariadne searches target providers in parallel. Each provider gets its own goroutine, while the search layers inside one provider still run in sequence.

That means the runtime shape is:

1. parse the source URL
2. fetch the source entity
3. search all target providers concurrently
4. within each provider, run identifier-first search layers in order
5. score the candidates returned by that provider

At the public API boundary, resolver methods return wrapped errors. Callers should use `errors.Is` with Ariadne's exported sentinels rather than comparing strings.

For the core maintained runtime adapters — Spotify, Apple Music, TIDAL, and SoundCloud — the current search contract is:

- skip malformed search hits before scoring
- keep already collected candidates when a later search layer fails
- keep already hydrated candidates when a later hydration fails
- only return the recorded search or hydration error when no usable candidates were recovered

This policy makes provider behavior more uniform without hiding service-specific credential or parse errors.

## Summary

| Service | Album source | Album target | Song source | Song target | Identifier support | Status |
|---|---|---|---|---|---|---|
| Spotify | Web API when credentials exist, otherwise public page bootstrap | Web API with app credentials | Web API when credentials exist, otherwise public page bootstrap | Web API with app credentials | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Apple Music | Public iTunes Lookup API | Public iTunes Search for metadata, official MusicKit for UPC/ISRC when `.p8` credentials exist | Public iTunes Lookup API | Public iTunes Search for metadata, official MusicKit for song ISRC when `.p8` credentials exist | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Deezer | Public Deezer album API | Public Deezer UPC lookup, track ISRC lookup, and metadata search | Public Deezer track API | Public Deezer track ISRC lookup and metadata search | album: UPC + ISRC + metadata; song: ISRC + metadata | supported |
| Bandcamp | Album page HTML + schema.org JSON-LD | Bandcamp search HTML + candidate hydration | Track page HTML + schema.org JSON-LD | Bandcamp search HTML + candidate hydration | metadata only | experimental |
| SoundCloud | Public set page hydration from `__sc_hydration` | Public-facing `api-v2` playlist search | Public track page hydration from `__sc_hydration` | Public-facing `api-v2` track search | metadata only in resolver | experimental |
| YouTube Music | Public album page HTML extraction | Public search HTML + candidate hydration | parse-only investigation so far; no runtime song adapter | no runtime song adapter | metadata only in resolver | experimental |
| TIDAL | Official catalog API with client credentials | Official metadata, UPC, and ISRC search with client credentials | Official catalog API with client credentials | Official song ISRC and metadata search with client credentials | album: UPC + ISRC + metadata; song: ISRC + metadata | experimental |
| Amazon Music | no runtime adapter | no runtime adapter | no runtime adapter | no runtime adapter | none | deferred |

## Spotify

### As an album source

- Supported input URLs look like `https://open.spotify.com/album/{id}`.
- Ariadne parses the album ID from the URL.
- If `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` are configured, source fetch prefers the official Spotify Web API.
- If those credentials are missing, Ariadne falls back to the public page bootstrap so Spotify can still be used as an input service.

### As an album target

Spotify album target search is enabled only when app credentials are configured. When it is enabled, Ariadne searches in this order:

1. album `UPC`
2. track `ISRC`
3. metadata search by album title and artist

Candidates are then reranked by the shared album scorer.

### As a song source

- Supported input URLs look like `https://open.spotify.com/track/{id}`.
- Song source fetch follows the same preference order as albums: Web API when credentials exist, otherwise public page bootstrap.

### As a song target

Spotify song target search is enabled only when app credentials are configured. When it is enabled, Ariadne searches in this order:

1. song `ISRC`
2. metadata search by song title and artist

Candidates are then reranked by the shared song scorer.

### Notes

- Spotify is one of the strongest services in Ariadne.
- Canonical album output URLs are rebuilt as `https://open.spotify.com/album/{id}`.
- Canonical song output URLs are rebuilt as `https://open.spotify.com/track/{id}`.

## Apple Music

### As an album source

- Supported input URLs are storefront-aware album URLs such as `https://music.apple.com/us/album/.../{id}`.
- Ariadne preserves the storefront from the input URL as a region hint.
- Source fetch uses the public iTunes Lookup API.

### As an album target

Apple Music always supports metadata search through the public iTunes Search API. Matching results are then hydrated through lookup before scoring.

If Apple Music credentials are configured:

- `APPLE_MUSIC_KEY_ID`
- `APPLE_MUSIC_TEAM_ID`
- `APPLE_MUSIC_PRIVATE_KEY_PATH`

Ariadne also enables official MusicKit identifier search by:

1. album `UPC`
2. track `ISRC`

Storefront matters. Ariadne uses the source storefront when present, otherwise the configured or default storefront.

### As a song source

- Supported input URLs are Apple Music track links represented as album-page URLs with a song query parameter, for example `https://music.apple.com/us/album/.../{album-id}?i={track-id}`.
- Ariadne preserves the storefront from the input URL as a region hint.
- Song source fetch also uses the public iTunes Lookup API.

### As a song target

Apple Music always supports metadata search for songs through the public iTunes Search API, then hydrates results through lookup before scoring.

If Apple Music credentials are configured, Ariadne also enables official MusicKit song lookup by `ISRC`.

### Notes

- Apple Music combines two runtime paths: public APIs for source fetch and metadata search, plus MusicKit for identifier search.
- Canonical album output URLs preserve storefront and album ID.
- Canonical song output URLs preserve storefront, album context, and the `?i={track-id}` query.

## Deezer

### As an album source

- Supported input URLs look like `https://www.deezer.com/album/{id}`.
- Ariadne normalizes region-prefixed Deezer links back to a regionless canonical album URL.
- Source fetch uses the public Deezer album API.

### As an album target

Deezer supports all three matching layers:

1. direct album lookup by `UPC`
2. track lookup by `ISRC`, followed by album hydration
3. metadata search by album title and artist

Candidates are reranked after hydration.

### As a song source

- Supported input URLs look like `https://www.deezer.com/track/{id}`.
- Source fetch uses the public Deezer track API.

### As a song target

Deezer song search uses:

1. direct track lookup by `ISRC`
2. metadata search by song title and artist

Candidates are reranked after hydration.

### Notes

- Deezer is one of the strongest low-friction services in the project.
- Canonical album output URLs are rebuilt as `https://www.deezer.com/album/{id}`.
- Canonical song output URLs are rebuilt as `https://www.deezer.com/track/{id}`.

## Bandcamp

### As an album source

- Supported input URLs are Bandcamp album pages such as `https://artist.bandcamp.com/album/{slug}`.
- Source fetch loads the album page and extracts schema.org JSON-LD from the HTML.

### As an album target

Bandcamp has no reliable identifier search. Ariadne instead:

1. searches `bandcamp.com/search?q=...`
2. extracts candidate album links from the HTML
3. hydrates those candidate album pages
4. reranks the hydrated results with the shared scorer

### As a song source

- Supported input URLs are Bandcamp track pages such as `https://artist.bandcamp.com/track/{slug}`.
- Source fetch loads the track page and extracts schema.org JSON-LD from the HTML.

### As a song target

Bandcamp song matching is metadata-first. Ariadne:

1. searches `bandcamp.com/search?q=...`
2. extracts track-result links from the HTML
3. hydrates those track pages
4. reranks the hydrated results with the shared song scorer

### Notes

- Bandcamp is useful for fuzzy matching, but it is still a scraping-based adapter.
- Canonical output URLs are the cleaned album or track page URLs.

## SoundCloud

### As an album source

- Supported input URLs are album-like set pages such as `https://soundcloud.com/{user}/sets/{slug}`.
- SoundCloud albums are really playlist or set resources rather than strong album objects.
- Source fetch loads the set page and extracts playlist data from `__sc_hydration`.

### As an album target

SoundCloud does not use release-level identifier search in Ariadne. Instead it uses:

1. metadata search against the public-facing `api-v2` playlist search
2. a discovered public `client_id` from the web app
3. shared scoring against playlist or set candidates

Some tracks expose UPC-like or ISRC-like publisher metadata, but Ariadne does not treat SoundCloud as a reliable identifier-first target.

### As a song source

- Supported input URLs are track pages such as `https://soundcloud.com/{user}/{slug}`.
- Source fetch loads the track page and extracts track data from `__sc_hydration`.

### As a song target

SoundCloud song resolution is also metadata-first. Ariadne uses:

1. metadata search against the public-facing `api-v2` track search
2. the discovered public `client_id` from the web app
3. shared song scoring against returned track candidates

### Notes

- SoundCloud is experimental because album semantics are weaker and the search path is unofficial.
- Canonical output URLs use the set or track `permalink_url`.

## YouTube Music

### As an album source

Supported input URLs include both:

- `https://music.youtube.com/browse/{browseId}`
- `https://music.youtube.com/playlist?list={playlistId}`

Source fetch uses public HTML extraction with a browser-like user-agent. Ariadne extracts title, canonical URL, artwork, artist, and track titles from inline page data.

### As an album target

YouTube Music does not use identifier search in Ariadne. Instead it:

1. searches public HTML for album-like candidates
2. extracts album browse IDs from search results
3. hydrates each candidate by loading its album page
4. scores the hydrated candidates with the shared matcher

### Song support

YouTube Music song resolution is still not implemented. Ariadne now parses common `music.youtube.com/watch?v=...` song URLs during evaluation work, but there is no stable runtime source or target adapter yet.

### Notes

- This adapter is experimental and somewhat brittle because it depends on public page structure instead of an official catalog API.
- Canonical output currently prefers the playlist or list URL exposed by the page.

## TIDAL

### As an album source

Supported input URLs include both:

- `https://tidal.com/album/{id}`
- `https://tidal.com/browse/album/{id}`

Source fetch uses the official TIDAL API and requires:

- `TIDAL_CLIENT_ID`
- `TIDAL_CLIENT_SECRET`

There is no public runtime fallback.

### As an album target

When credentials are configured, TIDAL album matching searches in this order:

1. album `UPC` / `barcodeId`
2. track `ISRC`
3. metadata search

All calls go through the official client-credentials token exchange and catalog API.

### As a song source

Supported input URLs include both:

- `https://tidal.com/track/{id}`
- `https://tidal.com/browse/track/{id}`

Song source fetch uses the same official TIDAL API and requires the same credentials.

### As a song target

When credentials are configured, TIDAL song matching searches in this order:

1. song `ISRC`
2. metadata search

### Notes

- TIDAL is technically strong, but still marked experimental because credentials are harder for typical OSS users to obtain.
- Canonical album output URLs are rebuilt as `https://tidal.com/album/{id}`.
- Canonical song output URLs are rebuilt as `https://tidal.com/track/{id}`.

## Amazon Music

### As an album source

- Ariadne recognizes Amazon Music album URLs such as `https://music.amazon.com/albums/{ASIN}`.
- Runtime source resolution is intentionally deferred after parsing.
- If you pass an Amazon Music URL as input, Ariadne returns a clear deferred-runtime error instead of pretending source fetch is supported.

### As an album target

Ariadne does not currently search Amazon Music as a target service.

### Song support

Amazon Music song resolution is not implemented.

### Notes

- Amazon Music remains deferred because the public pages are too thin and the practical API path is partner-gated.
- Current support is intentionally limited to URL recognition plus an explicit error path.

## Common resolver errors

If you are using the Go library, branch on exported errors with `errors.Is(...)` rather than string matching.

| Error | Meaning |
|---|---|
| `ariadne.ErrUnsupportedURL` | No registered source adapter recognized the input URL. |
| `ariadne.ErrNoSourceAdapters` | The resolver was built without any source adapters. |
| `ariadne.ErrAmazonMusicDeferred` | The input URL was recognized as Amazon Music, but runtime resolution is intentionally deferred. |
| `ariadne.ErrAppleMusicCredentialsNotConfigured` | An Apple Music official API operation required developer token credentials that were not configured. |
| `ariadne.ErrSpotifyCredentialsNotConfigured` | A Spotify Web API operation required app credentials that were not configured. |
| `ariadne.ErrTIDALCredentialsNotConfigured` | A TIDAL source or target operation required credentials that were not configured. |

## What the status labels mean

### Supported

- Ariadne can use the service in normal runtime flows, subject only to the documented credential requirements.
- Matching quality is good enough for core usage.

### Experimental

- Ariadne can resolve against the service today, but the runtime path depends on scraping, unofficial endpoints, weaker metadata, or a higher breakage risk.
- Matching quality is useful, but less dependable than the supported services.

### Deferred

- Ariadne does not ship a runtime adapter because the integration path is not credible enough yet for normal open-source use.
