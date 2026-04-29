# How Ariadne resolves albums and songs

This document shows runtime execution order for `ariadne resolve` and the resolver library.

Short version:

- parse the input URL
- fetch canonical metadata from the source service
- search every selected target service except the source service
- score and rank candidates
- optionally re-run Apple Music album search with identifiers learned from strong intermediate matches
- filter and format CLI output

Albums search by `UPC`, then track `ISRC`, then metadata. Songs search by `ISRC`, then metadata.

## CLI entrypoint

```text
ariadne resolve <url>
  |
  |-- load config file + environment + flags
  |-- parse resolve flags
  |     |-- --song     -> force song resolver
  |     |-- --album    -> force album resolver
  |     `-- neither    -> auto resolver
  |
  |-- build default resolver
  |     |-- build one adapter set per service
  |     |-- include target adapters allowed by --services / config
  |     |-- include Spotify targets only when Spotify credentials exist
  |     `-- include TIDAL targets only when TIDAL credentials exist
  |
  |-- create resolution timeout context
  |
  `-- execute resolver mode
        |-- song  -> ResolveSong(url)
        |-- album -> ResolveAlbum(url)
        `-- auto  -> Resolve(url)
```

Auto mode tries songs first and falls back to albums only for unsupported-song cases:

```text
Resolve(url)
  |
  |-- ResolveSong(url)
  |     |-- success -> return SongResolution
  |     |-- ErrUnsupportedURL / ErrNoSourceAdapters -> continue to album
  |     `-- any other error -> return error
  |
  `-- ResolveAlbum(url)
        |-- success -> return AlbumResolution
        `-- error   -> return error
```

Use `--album` when the input is definitely an album and you want to skip the song-parser pass.

## Source adapter parse order

Ariadne tests source adapters in a fixed order. First adapter that parses the URL owns the source fetch.

### Song source parse order

```text
1. Apple Music
2. Bandcamp
3. Deezer
4. SoundCloud
5. Spotify
6. TIDAL
7. YouTube Music
8. Amazon Music
```

### Album source parse order

```text
1. Apple Music
2. Deezer
3. Spotify
4. TIDAL
5. SoundCloud
6. YouTube Music
7. Amazon Music
8. Bandcamp
```

Amazon Music album and song URLs can parse, but runtime fetch is deferred and returns an error. YouTube Music song URLs can parse, but song fetch is also deferred.

## Album resolution order

```text
ResolveAlbum(url)
  |
  |-- parse source URL
  |     `-- first album source adapter that recognizes url wins
  |
  |-- fetch canonical source album
  |     `-- CanonicalAlbum{service, id, url, title, artists, release date,
  |                       UPC when known, track list, track ISRCs when known}
  |
  |-- choose target adapters
  |     |-- start from configured / default targets
  |     `-- remove source service from target list
  |
  |-- search targets concurrently
  |     |
  |     |-- target A
  |     |     |-- SearchByUPC(source.UPC)          if source UPC exists
  |     |     |-- SearchByISRC(source track ISRCs) if source ISRCs exist
  |     |     |-- SearchByMetadata(source)         always
  |     |     |-- dedupe candidates by service:id or service:url
  |     |     `-- score and rank candidates
  |     |
  |     |-- target B
  |     |     `-- same per-target sequence
  |     |
  |     `-- target N
  |           `-- same per-target sequence
  |
  |-- optional Apple Music cascade pass
  |     `-- see next section
  |
  `-- return Resolution{source, matches}
```

Inside one target, search layers run in order. Across targets, searches run concurrently, so no target can depend on another target until the optional Apple Music cascade pass.

## Apple Music album cascade pass

Apple Music public metadata search can miss albums that exist in the Apple catalog. To recover those cases, Ariadne can learn identifiers from other strong target matches and re-run Apple Music.

```text
Initial album target searches complete
  |
  |-- inspect non-Apple best matches
  |     `-- keep matches with score >= 100
  |
  |-- merge missing identifiers into source copy
  |     |-- copy UPC from strongest matches when source UPC is empty
  |     `-- copy track ISRCs from strongest matches when source tracks lack ISRCs
  |
  |-- identifiers changed?
  |     |-- no  -> keep initial Apple Music result
  |     `-- yes -> re-run Apple Music only
  |              |-- SearchByUPC(enriched.UPC)
  |              |-- SearchByISRC(enriched track ISRCs)
  |              |-- SearchByMetadata(enriched source)
  |              |-- score and rank using enriched source
  |              `-- replace appleMusic match
  |
  `-- final album matches
```

Example effect:

```text
Bandcamp source
  |
  |-- source has title + artist + tracks, but no UPC / ISRC
  |
  |-- Spotify / Deezer / TIDAL target search finds strong match
  |     `-- strong match has UPC + track ISRCs
  |
  |-- Apple Music cascade copies those identifiers
  |
  `-- Apple Music UPC / ISRC search finds exact album
```

Apple Music metadata fallback candidates are also pruned before they enter ranking:

```text
Apple Music metadata candidate
  |
  |-- score candidate against source
  |
  |-- score <= 0?
  |     `-- drop
  |
  |-- no title or artist evidence?
  |     |-- accepted evidence: title exact, core title, primary artist exact, artist overlap
  |     `-- drop
  |
  `-- keep candidate
```

This filter applies only to Apple Music metadata fallback candidates. Identifier candidates are not pruned by this rule.

## Song resolution order

```text
ResolveSong(url)
  |
  |-- parse source URL
  |     `-- first song source adapter that recognizes url wins
  |
  |-- fetch canonical source song
  |     `-- CanonicalSong{service, id, url, title, artists, duration,
  |                      ISRC when known, album info when known}
  |
  |-- choose target adapters
  |     |-- start from configured / default song targets
  |     `-- remove source service from target list
  |
  |-- search targets concurrently
  |     |
  |     |-- target A
  |     |     |-- SearchSongByISRC(source.ISRC) if source ISRC exists
  |     |     |-- SearchSongByMetadata(source) always
  |     |     |-- dedupe candidates by service:id or service:url
  |     |     `-- score and rank candidates
  |     |
  |     `-- target N
  |           `-- same per-target sequence
  |
  `-- return SongResolution{source, matches}
```

Song resolution does not currently run the Apple Music cascade pass.

YouTube Music and Amazon Music song URLs are recognized by the song parser, but their source fetch path is parse-only today. If one of those URLs is selected as the song source, resolution stops with a deferred-runtime error before target search.

## CLI output order

Library resolution returns all ranked matches. CLI output applies presentation rules after resolution:

```text
Resolution
  |
  |-- filter by --min-strength
  |     |-- very_weak -> keep all retained best matches
  |     |-- weak      -> keep score >= 50
  |     |-- probable  -> keep score >= 70
  |     `-- strong    -> keep score >= 100
  |
  |-- choose output shape
  |     |-- compact -> source link + best link per service
  |     `-- verbose -> source metadata + best + alternates + scores + reasons
  |
  `-- encode as json / yaml / csv
```

## Source-service fetch diagrams

Each source service produces the same canonical model shape, but the fetch path differs.

### Apple Music source

```text
Apple Music album URL
  |
  |-- parse storefront + album id
  |-- iTunes Lookup API: /lookup?id={albumID}&entity=song&country={storefront}
  |-- map collection + track rows
  `-- CanonicalAlbum
```

```text
Apple Music song URL
  |
  |-- parse storefront + track id from ?i={trackID}
  |-- iTunes Lookup API: /lookup?id={trackID}&entity=song&country={storefront}
  |-- pick song row
  `-- CanonicalSong
```

Apple Music source fetch uses public iTunes lookup. MusicKit credentials are used for identifier target search, not source fetch.

### Spotify source

```text
Spotify album URL
  |
  |-- parse album id
  |-- credentials configured?
  |     |-- yes -> Spotify Web API album fetch
  |     |          |-- if album not found -> public page bootstrap fallback
  |     |          `-- otherwise return API metadata
  |     `-- no  -> public page bootstrap fetch
  |
  `-- CanonicalAlbum
```

```text
Spotify track URL
  |
  |-- parse track id
  |-- Spotify Web API track fetch
  |     `-- requires SPOTIFY_CLIENT_ID + SPOTIFY_CLIENT_SECRET
  `-- CanonicalSong
```

Spotify target search always requires Spotify credentials.

### Deezer source

```text
Deezer album URL
  |
  |-- parse album id
  |-- Deezer public API album fetch
  |-- map album + tracks
  `-- CanonicalAlbum
```

```text
Deezer track URL
  |
  |-- parse track id
  |-- Deezer public API track fetch
  `-- CanonicalSong
```

Deezer does not require credentials.

### TIDAL source

```text
TIDAL album URL
  |
  |-- parse album id
  |-- TIDAL catalog API album fetch
  |     `-- requires TIDAL_CLIENT_ID + TIDAL_CLIENT_SECRET
  |-- fetch / map tracks
  `-- CanonicalAlbum
```

```text
TIDAL track URL
  |
  |-- parse track id
  |-- TIDAL catalog API track fetch
  |     `-- requires TIDAL_CLIENT_ID + TIDAL_CLIENT_SECRET
  `-- CanonicalSong
```

TIDAL source and target operations require credentials.

### Bandcamp source

```text
Bandcamp album page
  |
  |-- parse band / album slug
  |-- fetch album HTML
  |-- extract schema.org JSON-LD
  |-- map album + track list
  `-- CanonicalAlbum
```

```text
Bandcamp track page
  |
  |-- parse band / track slug
  |-- fetch track HTML
  |-- extract schema.org JSON-LD
  |-- map track + album info
  `-- CanonicalSong
```

Bandcamp has no reliable UPC / ISRC path, so Bandcamp source metadata usually drives metadata target search unless another target supplies identifiers for Apple Music cascade.

### SoundCloud source

```text
SoundCloud set URL
  |
  |-- parse user + set slug
  |-- fetch public page / hydration data
  |-- map set + tracks
  `-- CanonicalAlbum
```

```text
SoundCloud track URL
  |
  |-- parse user + track slug
  |-- fetch public page / hydration data
  `-- CanonicalSong
```

SoundCloud album support means playlist / set matching.

### YouTube Music source

```text
YouTube Music album-like URL
  |
  |-- parse browse / playlist form
  |-- fetch public HTML
  |-- extract album-like data
  `-- CanonicalAlbum
```

```text
YouTube Music watch URL
  |
  |-- parse video id from ?v={videoID}
  `-- stop: song runtime adapter is deferred
```

### Amazon Music source

```text
Amazon Music album URL
  |
  |-- parse album URL
  `-- stop: runtime adapter is deferred
```

```text
Amazon Music track URL
  |
  |-- parse /tracks/{trackASIN}
  `-- stop: runtime adapter is deferred
```

```text
Amazon Music album URL with trackAsin
  |
  |-- parse /albums/{albumASIN}?trackAsin={trackASIN}
  `-- stop: runtime adapter is deferred
```

Amazon Music is parse-only today.

## Target-service search diagrams

### Identifier-rich album targets

Spotify, Apple Music, Deezer, and TIDAL can use identifiers when available.

```text
Target album search
  |
  |-- UPC available?
  |     `-- SearchByUPC(upc)
  |
  |-- track ISRCs available?
  |     `-- SearchByISRC(isrcs)
  |
  |-- metadata fallback
  |     `-- SearchByMetadata(title + artists + other source metadata)
  |
  |-- hydrate candidates when search result is only a summary
  |-- dedupe candidates
  `-- score candidates
```

Service-specific notes:

```text
Spotify target      -> Web API search; credentials required
Apple Music target  -> MusicKit for UPC / ISRC when configured; public iTunes search for metadata
Deezer target       -> public UPC / ISRC lookup + public metadata search
TIDAL target        -> catalog API; credentials required
```

### Metadata-first album targets

Bandcamp, SoundCloud, and YouTube Music rely on metadata search.

```text
Target album search
  |
  |-- SearchByUPC skipped / returns none
  |-- SearchByISRC skipped / returns none
  |-- SearchByMetadata(source title + artist variants)
  |-- extract candidate links / IDs
  |-- hydrate candidate pages or API summaries
  |-- dedupe candidates
  `-- score candidates
```

### Song targets

```text
Target song search
  |
  |-- ISRC available?
  |     `-- SearchSongByISRC(isrc)
  |
  |-- metadata fallback
  |     `-- SearchSongByMetadata(title + artists + album info)
  |
  |-- hydrate candidates when needed
  |-- dedupe candidates
  `-- score candidates
```

## Scoring and ranking

Album candidates are scored from multiple signals:

```text
UPC exact
ISRC overlap
track-title overlap
title / core-title match
artist match / overlap
track-count match or mismatch
release-date / release-year match
duration near match
label exact
explicit mismatch penalty
edition mismatch / marker penalty
```

Song candidates use song-level signals:

```text
ISRC exact
title / core-title match
artist match / overlap
duration near match
album-title exact
release-date / release-year match
track-number exact
explicit mismatch penalty
edition mismatch / marker penalty
```

Ranking sorts candidates by descending score. Equal album scores break by candidate ID for stable output.

## Service support summary

| Service | Album input | Album target | Song input | Song target | Runtime search style | Notes |
|---|---:|---:|---:|---:|---|---|
| Apple Music | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | UPC / ISRC target search needs MusicKit credentials |
| Spotify | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | target search and song source need Spotify credentials |
| Deezer | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | no credentials required |
| TIDAL | Yes | Yes | Yes | Yes | album: UPC + ISRC + metadata; song: ISRC + metadata | source and target need TIDAL credentials |
| Bandcamp | Yes | Yes | Yes | Yes | metadata only | HTML / JSON-LD based |
| SoundCloud | Yes | Yes | Yes | Yes | metadata only | public page / API-v2 based |
| YouTube Music | Yes | Yes | Parse only | No | album metadata only | song fetch deferred |
| Amazon Music | Parse only | No | Parse only | No | none | runtime deferred |
