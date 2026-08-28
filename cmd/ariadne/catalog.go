package main

import (
	"strings"

	"github.com/xmbshwll/ariadne"
	"github.com/xmbshwll/ariadne/internal/config"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

// toWiringConfig converts the public config into the Provider Catalog's internal
// shape, applying the same trimming the library applies in New. It is the twin
// of package ariadne's internalConfig: both must normalize identically so a
// Catalog decision the CLI asks for matches what New would build.
func toWiringConfig(c ariadne.Config) config.Config {
	return config.Config{
		Spotify: config.Spotify{
			ClientID:     strings.TrimSpace(c.Spotify.ClientID),
			ClientSecret: strings.TrimSpace(c.Spotify.ClientSecret),
		},
		AppleMusic: config.AppleMusic{
			Storefront:     config.NormalizeStorefront(c.AppleMusicStorefront),
			KeyID:          strings.TrimSpace(c.AppleMusic.KeyID),
			TeamID:         strings.TrimSpace(c.AppleMusic.TeamID),
			PrivateKeyPath: strings.TrimSpace(c.AppleMusic.PrivateKeyPath),
		},
		TIDAL: config.TIDAL{
			ClientID:     strings.TrimSpace(c.TIDAL.ClientID),
			ClientSecret: strings.TrimSpace(c.TIDAL.ClientSecret),
		},
		HTTPTimeout: c.HTTPTimeout,
	}
}

// evaluateTarget asks the Provider Catalog whether a service can join Target
// Search for the given entity shape, under the public config.
func evaluateTarget(c ariadne.Config, name string, entity wiring.EntityShape) wiring.TargetServiceRequestDecision {
	return wiring.Default.EvaluateTarget(toWiringConfig(c), name, entity)
}

// targetServices lists the Catalog's target services for an entity shape.
func targetServices(c *ariadne.Config, entity wiring.EntityShape) []ariadne.ServiceName {
	if c == nil {
		return wiring.Default.TargetServices(nil, entity)
	}
	cfg := toWiringConfig(*c)
	return wiring.Default.TargetServices(&cfg, entity)
}

// describe reports the built-in support for one service.
func describe(service ariadne.ServiceName) (wiring.Capabilities, bool) {
	return wiring.Default.DescribeService(service)
}
