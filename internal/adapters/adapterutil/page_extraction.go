package adapterutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

var errRegexpGroupNotFound = errors.New("regexp group not found")

type PageRequest struct {
	BytesRequest
	Timeout time.Duration
}

type PageFetcher struct {
	Client         *http.Client
	UserAgent      string
	BuildError     string
	ExecuteError   string
	StatusError    StatusErrorFunc
	ErrorBodyLimit int64
	ReadError      string
	MaxBodyBytes   int64
	TooLargeError  error
	TooLarge       TooLargeErrorFunc
	Timeout        time.Duration
}

func (f PageFetcher) Fetch(ctx context.Context, requestURL string) ([]byte, error) {
	return FetchPage(ctx, PageRequest{
		BytesRequest: BytesRequest{
			RequestSpec: RequestSpec{
				Client:         f.Client,
				URL:            requestURL,
				UserAgent:      f.UserAgent,
				BuildError:     f.BuildError,
				ExecuteError:   f.ExecuteError,
				StatusError:    f.StatusError,
				ErrorBodyLimit: f.ErrorBodyLimit,
			},
			ReadError:     f.ReadError,
			MaxBodyBytes:  f.MaxBodyBytes,
			TooLargeError: f.TooLargeError,
			TooLarge:      f.TooLarge,
		},
		Timeout: f.Timeout,
	})
}

func FetchPage(ctx context.Context, spec PageRequest) ([]byte, error) {
	if spec.Timeout <= 0 {
		return FetchBytes(ctx, spec.BytesRequest)
	}

	timeout := spec.Timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			timeout = remaining
		}
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return FetchBytes(ctx, spec.BytesRequest)
}

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
