// Package htmlx pulls the data embedded inside a service HTML page: the JSON
// block a page assigns to a JavaScript global, rather than a document to parse.
package htmlx

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
)

var errRegexpGroupNotFound = errors.New("regexp group not found")

// FirstRegexpGroup returns the first capture group, or notFound when the page
// has no match (a changed page layout looks like this).
func FirstRegexpGroup(body []byte, pattern *regexp.Regexp, notFound error) ([]byte, error) {
	matches := pattern.FindSubmatch(body)
	if len(matches) < 2 {
		if notFound == nil {
			notFound = errRegexpGroupNotFound
		}
		return nil, notFound
	}
	return matches[1], nil
}

// DecodeJSONBlock decodes the JSON object a page assigns to a JavaScript global,
// reported as malformed when the page stops embedding it.
func DecodeJSONBlock[T any](
	body []byte,
	pattern *regexp.Regexp,
	notFound error,
	decodeError string,
	malformed error,
) (T, error) {
	var target T
	payload, err := FirstRegexpGroup(body, pattern, notFound)
	if err != nil {
		return target, err
	}
	if err := json.Unmarshal(payload, &target); err != nil {
		wrapped := fmt.Errorf("%s: %w", decodeError, err)
		if malformed != nil {
			return target, errors.Join(malformed, wrapped)
		}
		return target, wrapped
	}
	return target, nil
}
