package parseutil

import "strings"

// PathSegments splits a URL path into non-empty segments.
func PathSegments(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	segments := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			segments = append(segments, p)
		}
	}
	return segments
}

// IsRegionSegment returns true if the segment is a country code like "us" or "fr".
func IsRegionSegment(segment string) bool {
	if len(segment) != 2 {
		return false
	}
	for _, r := range segment {
		if r < 'a' || r > 'z' {
			return false
		}
	}
	return true
}
