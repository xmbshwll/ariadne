package main

import (
	"fmt"
	"strings"

	"github.com/xmbshwll/ariadne"
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
		decision := ariadne.EvaluateTargetServiceRequest(appConfig, part)
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

func targetServiceRequestError(raw string, decision ariadne.TargetServiceRequestDecision) error {
	if strings.TrimSpace(raw) == "" {
		return errNoTargetServicesSelected
	}
	switch decision.Status {
	case ariadne.TargetServiceRequestAvailable:
		return nil
	case ariadne.TargetServiceRequestParseOnly:
		return targetServiceDecisionError(errAmazonMusicTargetService, decision)
	case ariadne.TargetServiceRequestCredentialsRequired:
		return targetServiceCredentialError(decision.Service)
	default:
		return targetServiceDecisionError(errUnsupportedTargetService, decision)
	}
}

func targetServiceDecisionError(sentinel error, decision ariadne.TargetServiceRequestDecision) error {
	if decision.Message == "" {
		return sentinel
	}
	return fmt.Errorf("%w %s", sentinel, decision.Message)
}

func unsupportedTargetServiceError(raw string) error {
	decision := ariadne.EvaluateTargetServiceRequest(ariadne.Config{}, raw)
	return targetServiceDecisionError(errUnsupportedTargetService, decision)
}

func validateRequestedService(service ariadne.ServiceName, appConfig ariadne.Config) error {
	decision := ariadne.EvaluateConfiguredTargetService(appConfig, service)
	return targetServiceRequestError(string(service), decision)
}

func targetServiceCredentialError(service ariadne.ServiceName) error {
	switch service {
	case ariadne.ServiceSpotify:
		return errSpotifyTargetCredentials
	case ariadne.ServiceTIDAL:
		return errTIDALTargetCredentials
	default:
		return unsupportedTargetServiceError(string(service))
	}
}

func normalizeOutputFormat(raw string) (string, error) {
	format := strings.ToLower(strings.TrimSpace(raw))
	if format == "" {
		return outputFormatJSON, nil
	}
	if format != outputFormatJSON && format != outputFormatYAML && format != outputFormatCSV {
		return "", fmt.Errorf("%w %q (expected json, yaml, or csv)", errUnsupportedFormat, format)
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
		return "", fmt.Errorf("%w %q (expected very_weak, weak, probable, or strong)", errUnsupportedMinStrength, raw)
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
	for _, service := range ariadne.SupportedTargetServices() {
		names = appendUniqueServiceName(names, seen, string(service))
		capabilities, ok := ariadne.DescribeService(service)
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
