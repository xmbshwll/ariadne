package adapterutil

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	DefaultUserAgent = "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)"
	BrowserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36"
)

var errUnexpectedHTTPStatus = errors.New("unexpected http status")

type StatusErrorFunc func(statusCode int, body string) error

type TooLargeErrorFunc func(maxBytes int64) error

type RequestSpec struct {
	Client         *http.Client
	Method         string
	URL            string
	Body           io.Reader
	Headers        map[string]string
	UserAgent      string
	BuildError     string
	ExecuteError   string
	StatusError    StatusErrorFunc
	ErrorBodyLimit int64
}

type JSONRequest struct {
	RequestSpec
	DecodeError       string
	MalformedResponse error
}

type BytesRequest struct {
	RequestSpec
	ReadError     string
	MaxBodyBytes  int64
	TooLargeError error
	TooLarge      TooLargeErrorFunc
}

func StatusError(sentinel error) StatusErrorFunc {
	return func(statusCode int, body string) error {
		return fmt.Errorf("%w %d: %s", sentinel, statusCode, body)
	}
}

func GetJSON(ctx context.Context, spec JSONRequest, target any) error {
	resp, err := doRequest(ctx, spec.RequestSpec)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := statusError(resp, spec.RequestSpec); err != nil {
		return err
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		if spec.MalformedResponse != nil {
			return fmt.Errorf("%s: %w", spec.DecodeError, errors.Join(spec.MalformedResponse, err))
		}
		return fmt.Errorf("%s: %w", spec.DecodeError, err)
	}
	return nil
}

func FetchBytes(ctx context.Context, spec BytesRequest) ([]byte, error) {
	resp, err := doRequest(ctx, spec.RequestSpec)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := statusError(resp, spec.RequestSpec); err != nil {
		return nil, err
	}

	var reader io.Reader = resp.Body
	if spec.MaxBodyBytes > 0 {
		reader = io.LimitReader(resp.Body, spec.MaxBodyBytes+1)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", spec.ReadError, err)
	}
	if spec.MaxBodyBytes > 0 && len(body) > int(spec.MaxBodyBytes) {
		if spec.TooLarge != nil {
			return nil, spec.TooLarge(spec.MaxBodyBytes)
		}
		return nil, fmt.Errorf("%w: exceeded %d bytes", spec.TooLargeError, spec.MaxBodyBytes)
	}
	return body, nil
}

func doRequest(ctx context.Context, spec RequestSpec) (*http.Response, error) {
	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}
	req, err := http.NewRequestWithContext(ctx, method, spec.URL, spec.Body)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", spec.BuildError, err)
	}
	if spec.UserAgent != "" {
		req.Header.Set("User-Agent", spec.UserAgent)
	}
	for name, value := range spec.Headers {
		req.Header.Set(name, value)
	}
	client := spec.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", spec.ExecuteError, err)
	}
	return resp, nil
}

func statusError(resp *http.Response, spec RequestSpec) error {
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return nil
	}
	limit := spec.ErrorBodyLimit
	if limit == 0 {
		limit = 4096
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, limit))
	message := strings.TrimSpace(string(body))
	if spec.StatusError != nil {
		return spec.StatusError(resp.StatusCode, message)
	}
	return fmt.Errorf("%w %d: %s", errUnexpectedHTTPStatus, resp.StatusCode, message)
}
