# How album resolution works by service

This document describes the **current runtime resolution path** for every service in Ariadne.

The shared resolver flow is always:

1. parse the input album URL
2. fetch one canonical source album from the source service
3. search every other target service in this order:
   - `UPC`
   - track `ISRC`
   - metadata (`album title + primary artist`)
4. deduplicate candidates
5. score candidates using shared matching logic
6. return the best match and alternates per service

Not every service supports every search step. The sections below explain the real path Ariadne uses today.

## Summary

| Service | Source fetch path | Target search path | Identifier support | Runtime status |
|---|---|---|---|---|
| Spotify | Web API when creds exist, otherwise public page bootstrap | Web API with app credentials | UPC + ISRC + metadata | supported |
| Apple Music | Public iTunes Lookup API | Public iTunes Search for metadata, official MusicKit for UPC/ISRC when `.p8` creds exist | UPC + ISRC + metadata | supported |
| Deezer | Public Deezer album API | Public Deezer UPC lookup, track ISRC lookup, and metadata search | UPC + ISRC + metadata | supported |
| Bandcamp | Album page HTML + schema.org JSON-LD | Bandcamp search HTML + candidate hydration | metadata only | experimental |
| SoundCloud | Public set page hydration from `__sc_hydration` | Public-facing `api-v2` playlist search | metadata only in resolver | experimental |
| YouTube Music | Public album page HTML extraction | Public search HTML + candidate hydration | metadata only in resolver | experimental |
| TIDAL | Official catalog API with client credentials | Official metadata, UPC, and ISRC search with client credentials | UPC + ISRC + metadata | experimental |
| Amazon Music | no runtime adapter | no runtime adapter | none | deferred |

---

## Spotify

### Input/source resolution
- Accepted input URLs are Spotify album URLs such as `https://open.spotify.com/album/{id}`.
- Ariadne parses the album ID from the URL.
- If `SPOTIFY_CLIENT_ID` and `SPOTIFY_CLIENT_SECRET` are configured, source fetch prefers the official Spotify Web API.
- If those credentials are missing, source fetch falls back to the public page bootstrap so Spotify can still be used as an input service.

### Target resolution
- Spotify target matching is enabled only when app credentials are configured.
- The resolver tries Spotify in this order:
  1. album `UPC`
  2. track `ISRC`
  3. metadata search by album title + artist
- Candidates are then reranked by the shared scorer.

### Practical notes
- Spotify is one of the strongest services in Ariadne because it has reliable identifiers and structured metadata.
- Canonical output URLs are rebuilt as `https://open.spotify.com/album/{id}`.

---

## Apple Music

### Input/source resolution
- Accepted input URLs are storefront-aware album URLs such as `https://music.apple.com/us/album/.../{id}`.
- Ariadne preserves the storefront from the input URL as `RegionHint`.
- Source fetch uses the public iTunes Lookup API today.

### Target resolution
- Apple Music target matching always supports metadata search through the public iTunes Search API.
- Ariadne hydrates matching search results through lookup before scoring them.
- If Apple Music `.p8` credentials are configured:
  - `APPLE_MUSIC_KEY_ID`
  - `APPLE_MUSIC_TEAM_ID`
  - `APPLE_MUSIC_PRIVATE_KEY_PATH`
  then Ariadne also enables official MusicKit target search by:
  1. album `UPC`
  2. track `ISRC`
- Storefront matters. Ariadne uses the source storefront when present, otherwise the configured/default storefront.

### Practical notes
- Apple Music is a first-class connector, but it mixes two runtime paths:
  - public APIs for source fetch and metadata search
  - official MusicKit auth for identifier search
- Canonical output URLs preserve storefront and album ID.

---

## Deezer

### Input/source resolution
- Accepted input URLs are Deezer album URLs such as `https://www.deezer.com/album/{id}`.
- Ariadne normalizes region-prefixed Deezer links back to a regionless canonical album URL.
- Source fetch uses the public Deezer album API.

### Target resolution
- Deezer target matching supports all three layers:
  1. direct album lookup by `UPC`
  2. track lookup by `ISRC`, then album hydration
  3. metadata search by album title + artist
- Candidates are reranked by the shared scorer after hydration.

### Practical notes
- Deezer is one of the best public, low-friction connectors in the project.
- Canonical output URLs are rebuilt as `https://www.deezer.com/album/{id}`.

---

## Bandcamp

### Input/source resolution
- Accepted input URLs are Bandcamp album pages such as `https://artist.bandcamp.com/album/{slug}`.
- Source fetch loads the album page and extracts schema.org JSON-LD from the HTML.

### Target resolution
- Bandcamp has no reliable identifier search.
- Ariadne resolves Bandcamp targets by:
  1. searching `bandcamp.com/search?q=...`
  2. extracting candidate album links from the HTML
  3. hydrating those candidate album pages
  4. reranking hydrated results with the shared scorer

### Practical notes
- Bandcamp works well enough for fuzzy matching, but it is still a scraping-based adapter.
- Canonical output URLs are the cleaned album page URLs.

---

## SoundCloud

### Input/source resolution
- Accepted input URLs are album-like set pages such as `https://soundcloud.com/{user}/sets/{slug}`.
- SoundCloud albums are really playlist/set resources, not strong album objects.
- Source fetch loads the set page and extracts the playlist payload from `__sc_hydration`.

### Target resolution
- SoundCloud does not use release-level identifier search in Ariadne.
- Target matching uses:
  1. metadata search against the public-facing `api-v2` playlist search
  2. a discovered public `client_id` from the web app
  3. shared scoring against playlist/set candidates
- Track-level publisher metadata can still improve fuzzy matching because some tracks expose UPC/ISRC-like fields, but the resolver does not treat SoundCloud as a reliable identifier-first target.

### Practical notes
- SoundCloud is explicitly experimental because album semantics are weaker and the web-facing search path is unofficial.
- Canonical output URLs use the playlist/set `permalink_url`.

---

## YouTube Music

### Input/source resolution
- Accepted input URLs include both:
  - `https://music.youtube.com/browse/{browseId}`
  - `https://music.youtube.com/playlist?list={playlistId}`
- Source fetch uses public HTML extraction with a browser-like user-agent.
- Ariadne extracts:
  - title
  - canonical playlist/list URL
  - artwork
  - artist
  - track titles from inline page data

### Target resolution
- YouTube Music does not use identifier search in Ariadne.
- Target matching uses:
  1. public search HTML for album-like candidates
  2. extraction of album browse IDs from search results
  3. hydration of each candidate by loading its album page
  4. shared scoring against hydrated candidates

### Practical notes
- This adapter is experimental and brittle because it depends on public page structure rather than an official catalog API.
- Canonical output currently prefers the playlist/list URL exposed by the page.

---

## TIDAL

### Input/source resolution
- Accepted input URLs include both canonical and browse forms such as:
  - `https://tidal.com/album/{id}`
  - `https://tidal.com/browse/album/{id}`
- Source fetch uses the official TIDAL API.
- This requires:
  - `TIDAL_CLIENT_ID`
  - `TIDAL_CLIENT_SECRET`
- There is no public runtime fallback.

### Target resolution
- TIDAL target matching is fully identifier-aware when credentials are configured.
- Ariadne resolves TIDAL targets in this order:
  1. album `UPC` / `barcodeId`
  2. track `ISRC`
  3. metadata search
- All calls go through the official client-credentials token exchange and catalog API.

### Practical notes
- TIDAL is technically strong, but still experimental because normal OSS users may not find credentials as easy to obtain as Spotify/Apple/Deezer credentials.
- Canonical output URLs are rebuilt as `https://tidal.com/album/{id}`.

---

## Amazon Music

### Input/source resolution
- Ariadne now recognizes Amazon Music album URLs such as `https://music.amazon.com/albums/{ASIN}`.
- The runtime source adapter is intentionally deferred after parsing.
- If an Amazon Music album URL is used as input, Ariadne returns a clear deferred-runtime error instead of pretending that source metadata fetch is supported.

### Target resolution
- Ariadne does not currently search Amazon Music as a target service.

### Practical notes
- Amazon Music remains deferred because the public pages are too thin and the documented API path is closed beta / partner-gated.
- The current runtime behavior is intentionally limited to URL recognition plus an explicit error path.

---

## What “supported”, “experimental”, and “deferred” mean in resolver terms

### Supported
- Ariadne can use the service in normal runtime flows without hidden caveats beyond documented credentials.
- Matching quality is good enough for core usage.

### Experimental
- Ariadne can resolve against the service now, but the runtime path depends on scraping, unofficial endpoints, weaker metadata, or higher breakage risk.
- Matching quality is useful, but not as dependable as the supported connectors.

### Deferred
- Ariadne does not ship a runtime adapter because the integration path is not yet credible for normal OSS use.
