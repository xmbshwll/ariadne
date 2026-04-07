// Package ariadne resolves album URLs across music services.
//
// Given one supported album URL, Ariadne fetches canonical metadata from the
// source service, searches configured target services, and returns ranked
// matches.
//
// Typical usage is:
//   - create a Config with DefaultConfig or LoadConfig
//   - build a Resolver with New
//   - call ResolveAlbum with an album URL from a supported source service
//
// The CLI in cmd/ariadne is a thin wrapper around this package.
package ariadne
