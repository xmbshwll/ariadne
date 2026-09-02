// Package ariadne resolves album and song URLs across music services.
//
// Given one supported album or song URL, Ariadne fetches canonical metadata
// from the source service, searches configured target services, and returns
// ranked matches.
//
// Typical usage is:
//   - create a Config with DefaultConfig or LoadConfigFromEnv
//   - build a Resolver with New
//   - call ResolveAlbum, ResolveSong, or Resolve depending on your input shape
//
// The CLI in cmd/ariadne is a thin wrapper around this package.
//
// This package is the resolve surface. Service capability and
// adapter-construction decisions live in the Provider Catalog (internal/wiring);
// entity resolution lives in internal/resolve. The CLI in cmd/ariadne reaches
// the Catalog directly through internal/wiring rather than through this package.
package ariadne
