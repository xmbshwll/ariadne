package main

// The Provider Catalog seam for the CLI: config conversion, catalog queries, target-service validation, and its error mapping.

import (
	"fmt"
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
	credentials := config.NormalizeCredentials(config.CredentialsShape{
		SpotifyClientID:          c.Spotify.ClientID,
		SpotifyClientSecret:      c.Spotify.ClientSecret,
		AppleMusicKeyID:          c.AppleMusic.KeyID,
		AppleMusicTeamID:         c.AppleMusic.TeamID,
		AppleMusicPrivateKeyPath: c.AppleMusic.PrivateKeyPath,
		AppleMusicStorefront:     c.AppleMusicStorefront,
		TIDALClientID:            c.TIDAL.ClientID,
		TIDALClientSecret:        c.TIDAL.ClientSecret,
	})
	return config.Config{
		Spotify: config.Spotify{
			ClientID:     credentials.SpotifyClientID,
			ClientSecret: credentials.SpotifyClientSecret,
		},
		AppleMusic: config.AppleMusic{
			Storefront:     credentials.AppleMusicStorefront,
			KeyID:          credentials.AppleMusicKeyID,
			TeamID:         credentials.AppleMusicTeamID,
			PrivateKeyPath: credentials.AppleMusicPrivateKeyPath,
		},
		TIDAL: config.TIDAL{
			ClientID:     credentials.TIDALClientID,
			ClientSecret: credentials.TIDALClientSecret,
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

// services lists every Music Service the Provider Catalog knows, in order.
func services() []ariadne.ServiceName {
	return wiring.Default.Services()
}

// helpServiceNotes renders the song-resolution notes: the hydrating services
// from the Catalog's song targets, then one deferred line per Music Service
// that parses song Source Input without a runtime song Target Search role.
func helpServiceNotes() []string {
	notes := make([]string, 0)
	for _, service := range services() {
		capabilities, ok := describe(service)
		if ok && capabilities.SupportsRuntimeSongInputURL && !capabilities.SupportsSongTarget {
			notes = append(notes, fmt.Sprintf("  - %s song URLs are recognized, but runtime hydration remains deferred.", service))
		}
	}
	return notes
}

// helpSongHydrationNote lists the Music Services whose song Runtime Hydration
// is available, straight from the Provider Catalog's song targets.
func helpSongHydrationNote() string {
	services := serviceNames(targetServices(nil, wiring.EntityShapeSong))
	return fmt.Sprintf("  - Song resolution currently hydrates %s.", strings.Join(services, ", "))
}

// helpServiceCaveats renders the --services caveats: one line per target
// service whose request under an empty config is not plainly available, in
// Catalog order.
func helpServiceCaveats() string {
	caveats := make([]string, 0)
	for _, service := range targetServices(nil, wiring.EntityShapeAny) {
		decision := evaluateTarget(ariadne.Config{}, string(service), wiring.EntityShapeAny)
		switch decision.Status {
		case wiring.TargetServiceRequestCredentialsRequired:
			caveats = append(caveats, fmt.Sprintf("      %s target search requires credentials: %s.", service, decision.CredentialHint))
		case wiring.TargetServiceRequestParseOnly:
			caveats = append(caveats, fmt.Sprintf("      %s is not available as a target service.", service))
		}
	}
	return strings.Join(caveats, "\n")
}

func parseRequestedServices(raw string, appConfig ariadne.Config) ([]ariadne.ServiceName, error) {
	if strings.TrimSpace(raw) == "" {
		services := append([]ariadne.ServiceName(nil), appConfig.TargetServices...)
		for _, service := range services {
			if err := validateRequestedService(service, appConfig); err != nil {
				return nil, err
			}
		}
		return services, nil
	}

	services := make([]ariadne.ServiceName, 0)
	seen := map[ariadne.ServiceName]struct{}{}
	for part := range strings.SplitSeq(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		decision := evaluateTarget(appConfig, part, wiring.EntityShapeAny)
		if err := targetServiceRequestError(part, decision); err != nil {
			return nil, err
		}
		service := decision.Service
		if _, ok := seen[service]; ok {
			continue
		}
		seen[service] = struct{}{}
		services = append(services, service)
	}
	if len(services) == 0 {
		return nil, errNoTargetServicesSelected
	}
	return services, nil
}

func targetServiceRequestError(raw string, decision wiring.TargetServiceRequestDecision) error {
	if strings.TrimSpace(raw) == "" {
		return errNoTargetServicesSelected
	}
	switch decision.Status {
	case wiring.TargetServiceRequestAvailable:
		return nil
	case wiring.TargetServiceRequestParseOnly:
		return targetServiceDecisionError(errAmazonMusicTargetService, decision)
	case wiring.TargetServiceRequestCredentialsRequired:
		return targetServiceCredentialError(decision)
	default:
		return targetServiceDecisionError(errUnsupportedTargetService, decision)
	}
}

// cliError reports a sentinel to errors.Is while keeping the full user-facing
// text. main.rootError prints the innermost error of a chain, so an explanation
// built with %w would be unwrapped away before it reaches stderr.
type cliError struct {
	sentinel error
	message  string
}

func (e cliError) Error() string { return e.message }

func (e cliError) Is(target error) bool { return target == e.sentinel }

// withDetail composes one sentinel with a detail phrase without hiding it behind
// the sentinel when the CLI prints the error. The format owns its own separator.
func withDetail(sentinel error, format string, args ...any) error {
	return cliError{sentinel: sentinel, message: fmt.Sprintf(sentinel.Error()+format, args...)}
}

func targetServiceDecisionError(sentinel error, decision wiring.TargetServiceRequestDecision) error {
	if decision.Message == "" {
		return sentinel
	}
	return withDetail(sentinel, " %s", decision.Message)
}

func targetServiceCredentialError(decision wiring.TargetServiceRequestDecision) error {
	return withDetail(errTargetServiceCredentials, ": %s", decision.CredentialHint)
}

func validateRequestedService(service ariadne.ServiceName, appConfig ariadne.Config) error {
	decision := evaluateTarget(appConfig, string(service), wiring.EntityShapeAny)
	return targetServiceRequestError(string(service), decision)
}

func normalizeOutputFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "" {
		return outputFormatJSON, nil
	}
	if format != outputFormatJSON && format != outputFormatYAML && format != outputFormatCSV {
		return "", withDetail(errUnsupportedFormat, " %q (expected json, yaml, or csv)", format)
	}
	return format, nil
}

func parseMatchStrength(raw string) (ariadne.MatchStrength, error) {
	normalized := normalizeLookupKey(raw)
	if normalized == "" {
		return ariadne.MatchStrengthVeryWeak, nil
	}
	strength, ok := matchStrengthByName[normalized]
	if !ok {
		return "", withDetail(errUnsupportedMinStrength, " %q (expected very_weak, weak, probable, or strong)", raw)
	}
	return strength, nil
}

func normalizeLookupKey(raw string) string {
	return valueNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

func serviceNames(services []ariadne.ServiceName) []string {
	names := make([]string, 0, len(services))
	for _, service := range services {
		names = append(names, string(service))
	}
	return names
}

func targetServiceNamesUsage() string {
	names := make([]string, 0)
	seen := map[string]struct{}{}
	for _, service := range targetServices(nil, wiring.EntityShapeAny) {
		names = appendUniqueServiceName(names, seen, string(service))
		capabilities, ok := describe(service)
		if !ok {
			continue
		}
		for _, alias := range capabilities.Aliases {
			names = appendUniqueServiceName(names, seen, alias)
		}
	}
	return strings.Join(names, ", ")
}

func appendUniqueServiceName(names []string, seen map[string]struct{}, name string) []string {
	if name == "" {
		return names
	}
	if _, ok := seen[name]; ok {
		return names
	}
	seen[name] = struct{}{}
	return append(names, name)
}
