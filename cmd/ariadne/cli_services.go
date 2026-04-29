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
		service, err := normalizeRequestedService(part)
		if err != nil {
			return nil, err
		}
		if err := validateRequestedService(service, appConfig); err != nil {
			return nil, err
		}
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

func normalizeRequestedService(raw string) (ariadne.ServiceName, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errNoTargetServicesSelected
	}

	service, ok := ariadne.LookupServiceName(raw)
	if !ok {
		return "", unsupportedTargetServiceError(raw)
	}
	if service == ariadne.ServiceAmazonMusic {
		return "", errAmazonMusicTargetService
	}
	if !ariadne.SupportsTarget(service) {
		return "", unsupportedTargetServiceError(raw)
	}
	return service, nil
}

func unsupportedTargetServiceError(raw string) error {
	return fmt.Errorf("%w %q (expected one of the supported target services: %s)", errUnsupportedTargetService, raw, strings.Join(serviceNames(ariadne.SupportedTargetServices()), ", "))
}

func validateRequestedService(service ariadne.ServiceName, appConfig ariadne.Config) error {
	if ariadne.SupportsEnabledTarget(appConfig, service) {
		return nil
	}
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
