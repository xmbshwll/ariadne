package adapterutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var errRegexpGroupNotFound = errors.New("regexp group not found")

type PageRequest struct {
	BytesRequest
	Timeout time.Duration
}

func FetchPage(ctx context.Context, spec PageRequest) ([]byte, error) {
	if spec.Timeout <= 0 {
		return FetchBytes(ctx, spec.BytesRequest)
	}
	if _, ok := ctx.Deadline(); ok {
		return FetchBytes(ctx, spec.BytesRequest)
	}
	ctx, cancel := context.WithTimeout(ctx, spec.Timeout)
	defer cancel()
	return FetchBytes(ctx, spec.BytesRequest)
}

func FirstRegexpGroup(body []byte, pattern *regexp.Regexp, notFound error) ([]byte, error) {
	matches := pattern.FindSubmatch(body)
	if len(matches) != 2 {
		if notFound == nil {
			notFound = errRegexpGroupNotFound
		}
		return nil, notFound
	}
	return matches[1], nil
}

func DecodeJSONBlock[T any](body []byte, pattern *regexp.Regexp, notFound error, decodeError string, malformed error) (T, error) {
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
