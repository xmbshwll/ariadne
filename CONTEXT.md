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

**Runtime Hydration**:
The network-backed step that turns a parsed Music Service URL into canonical album or song metadata.
_Avoid_: fetch, scrape

**Target Search**:
The step that searches a Music Service for candidates matching a hydrated source entity.
_Avoid_: lookup, discovery

## Relationships

- A **Provider Catalog** contains one entry per built-in **Music Service**.
- A **Music Service** exposes zero or more **Capabilities**.
- **Runtime Hydration** is required for source **Capabilities** but can be intentionally deferred for parse-only **Music Services**.
- **Target Search** is required for target **Capabilities**.

## Example dialogue

> **Dev:** "Does **Amazon Music** have a song source **Capability**?"
> **Domain expert:** "It can parse the URL, but **Runtime Hydration** is deferred, so the **Provider Catalog** must report that constraint explicitly."

## Flagged ambiguities

- "provider" and "service" both referred to external music platforms — resolved: use **Music Service** for the platform and **Provider Catalog** for Ariadne's catalog Module.
