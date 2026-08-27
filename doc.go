// Package ariadne resolves album and song URLs across music services.
//
// Given one supported album or song URL, Ariadne fetches canonical metadata
// from the source service, searches configured target services, and returns
// ranked matches.
//
// Typical usage is:
//   - create a Config with DefaultConfig or LoadConfig
//   - build a Resolver with New
//   - call ResolveAlbum, ResolveSong, or Resolve depending on your input shape
//
// The CLI in cmd/ariadne is a thin wrapper around this package.
//
// This package is a facade. Service capability and adapter-construction
// decisions live in the Provider Catalog (internal/wiring); entity resolution
// lives in internal/resolve. Everything here is a re-export or a thin wrapper.
package ariadne
