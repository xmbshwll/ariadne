package services

import (
	"strings"

	"github.com/xmbshwll/ariadne/internal/model"
)

type definition struct {
	name           model.ServiceName
	aliases        []string
	supportsTarget bool
}

var lookupKeyNormalizer = strings.NewReplacer("-", "", "_", "")

var definitions = []definition{
	{name: model.ServiceAppleMusic, aliases: []string{"applemusic"}, supportsTarget: true},
	{name: model.ServiceBandcamp, aliases: []string{"bandcamp"}, supportsTarget: true},
	{name: model.ServiceDeezer, aliases: []string{"deezer"}, supportsTarget: true},
	{name: model.ServiceSoundCloud, aliases: []string{"soundcloud"}, supportsTarget: true},
	{name: model.ServiceSpotify, aliases: []string{"spotify"}, supportsTarget: true},
	{name: model.ServiceTIDAL, aliases: []string{"tidal"}, supportsTarget: true},
	{name: model.ServiceYouTubeMusic, aliases: []string{"youtubemusic", "ytmusic"}, supportsTarget: true},
	{name: model.ServiceAmazonMusic, aliases: []string{"amazonmusic", "amazon"}},
}

var (
	servicesByLookupKey       = buildLookup(definitions, func(def definition) bool { return true })
	targetServicesByLookupKey = buildLookup(definitions, func(def definition) bool { return def.supportsTarget })
	aliasesByService          = buildAliasesByService(definitions)
)

func Lookup(raw string) (model.ServiceName, bool) {
	service, ok := servicesByLookupKey[NormalizeLookupKey(raw)]
	return service, ok
}

func LookupTarget(raw string) (model.ServiceName, bool) {
	service, ok := targetServicesByLookupKey[NormalizeLookupKey(raw)]
	return service, ok
}

func AliasesFor(service model.ServiceName) []string {
	aliases := aliasesByService[service]
	if len(aliases) == 0 {
		return nil
	}
	return append([]string(nil), aliases...)
}

func NormalizeLookupKey(raw string) string {
	return lookupKeyNormalizer.Replace(strings.ToLower(strings.TrimSpace(raw)))
}

func buildLookup(defs []definition, include func(definition) bool) map[string]model.ServiceName {
	lookup := make(map[string]model.ServiceName, len(defs)*2)
	for _, def := range defs {
		if !include(def) {
			continue
		}
		lookup[NormalizeLookupKey(string(def.name))] = def.name
		for _, alias := range def.aliases {
			lookup[NormalizeLookupKey(alias)] = def.name
		}
	}
	return lookup
}

func buildAliasesByService(defs []definition) map[model.ServiceName][]string {
	aliasesByService := make(map[model.ServiceName][]string, len(defs))
	for _, def := range defs {
		aliasesByService[def.name] = append([]string(nil), def.aliases...)
	}
	return aliasesByService
}
