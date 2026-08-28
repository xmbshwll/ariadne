package main

import (
	"fmt"
	"strings"

	"github.com/xmbshwll/ariadne"
	"github.com/xmbshwll/ariadne/internal/wiring"
)

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
