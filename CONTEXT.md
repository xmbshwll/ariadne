# Ariadne Context

Ariadne resolves music URLs across external music services by parsing source URLs, hydrating canonical metadata, and searching target services for matching entities.

## Language

**Music Service**:
An external music platform Ariadne can recognize, hydrate from, or search against.
_Avoid_: provider, backend

**Provider Catalog**:
The authoritative catalog of Ariadne's built-in Music Services, their capabilities, runtime constraints, ordering, and Adapter wiring.
_Avoid_: registry, service list

**Capability**:
A supported resolution role for a Music Service, such as album source, album target, song source, or song target.
_Avoid_: feature flag, support bit

**Source Input**:
A user-supplied music URL while Ariadne is recognizing its Music Service and entity shape.
_Avoid_: raw URL, input string

**Runtime Hydration**:
The network-backed step that turns a parsed Music Service URL into canonical album or song metadata.
_Avoid_: fetch, scrape

**Target Search**:
The step that searches a Music Service for candidates matching a hydrated source entity.
_Avoid_: lookup, discovery

**Identifier Enrichment**:
The repair step that copies missing UPC or ISRC identifiers from strong intermediate matches into a source copy for a follow-up Target Search.
_Avoid_: cascade hack, backfill

**Credential Token**:
A short-lived access token issued from configured Music Service credentials and cached for credentialed Runtime Hydration or Target Search.
_Avoid_: auth blob, bearer cache

**Entity Resolution**:
The end-to-end pipeline that recognizes Source Input, performs Runtime Hydration, runs Target Search, and returns ranked matches for one music entity shape.
_Avoid_: resolver orchestration, flow glue

## Relationships

- A **Provider Catalog** contains one entry per built-in **Music Service**.
- A **Music Service** exposes zero or more **Capabilities**.
- A **Source Input** resolves to at most one **Music Service** before **Runtime Hydration** starts.
- **Runtime Hydration** is required for source **Capabilities** but can be intentionally deferred for parse-only **Music Services**.
- **Target Search** is required for target **Capabilities**.
- **Identifier Enrichment** can trigger a follow-up **Target Search** for a **Music Service** whose metadata search needs stronger identifiers.
- A **Credential Token** is required only by Music Services whose source or target **Capabilities** need credentialed network access.
- **Entity Resolution** composes Source Input recognition, Runtime Hydration, Target Search, and optional Identifier Enrichment.

## Example dialogue

> **Dev:** "What happens if a **Source Input** looks like Spotify but cannot be hydrated?"
> **Domain expert:** "Recognition succeeded, then **Runtime Hydration** explains the failure for that **Music Service**."
>
> **Dev:** "Does **Amazon Music** have a song source **Capability**?"
> **Domain expert:** "It can parse the URL, but **Runtime Hydration** is deferred, so the **Provider Catalog** must report that constraint explicitly."
>
> **Dev:** "Why does Apple Music run another **Target Search** after Spotify matches?"
> **Domain expert:** "That is **Identifier Enrichment**: Spotify can provide UPC or ISRC evidence that Apple Music metadata search could not see."

## Flagged ambiguities

- "provider" and "service" both referred to external music platforms — resolved: use **Music Service** for the platform and **Provider Catalog** for Ariadne's catalog Module.
