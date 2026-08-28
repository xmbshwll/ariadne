package spotify

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/auth"
	"github.com/xmbshwll/ariadne/internal/httpx"
)

const (
	spotifyTokenRefreshTimeout = 30 * time.Second
	spotifyAPIMaxAttempts      = 6
	spotifyAPIRetryBackoff     = 250 * time.Millisecond
)

type spotifyAPIError struct {
	StatusCode int
	Message    string
}

func (e *spotifyAPIError) Error() string {
	return fmt.Sprintf("%s %d: %s", errUnexpectedSpotifyAPIStatus.Error(), e.StatusCode, e.Message)
}

func (e *spotifyAPIError) Is(target error) bool {
	return target == errUnexpectedSpotifyAPIStatus
}

func (e *spotifyAPIError) HTTPStatusCode() int {
	return e.StatusCode
}

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	err := httpx.Retry(ctx, spotifyAPIMaxAttempts, spotifyAPIRetryBackoff, func(ctx context.Context) error {
		return a.getAPIJSONOnce(ctx, endpoint, target)
	})
	if err != nil {
		return fmt.Errorf("spotify api request: %w", err)
	}
	return nil
}

func (a *Adapter) getAPIJSONOnce(ctx context.Context, endpoint string, target any) error {
	token, err := a.AccessToken(ctx)
	if err != nil {
		return err
	}

	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       a.client,
			URL:          endpoint,
			Headers:      map[string]string{"Authorization": "Bearer " + token},
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build api request",
			ExecuteError: "execute api request",
			StatusError: func(statusCode int, body string) error {
				return &spotifyAPIError{StatusCode: statusCode, Message: body}
			},
		},
		DecodeError:       "decode api response",
		MalformedResponse: ErrMalformedSpotifyAPIResponse,
	}, target)
}

func (a *Adapter) AccessToken(ctx context.Context) (string, error) {
	//nolint:wrapcheck // Credential token source preserves service-specific token errors.
	return a.tokenSource.AccessToken(ctx)
}

func (a *Adapter) newTokenSource() *auth.TokenSource {
	return auth.NewTokenSource(auth.TokenSourceConfig{
		Service: "%s",
		Credentials: auth.ClientCredentials{
			ClientID:     a.clientID,
			ClientSecret: a.clientSecret,
		},
		MissingCredentials: ErrCredentialsNotConfigured,
		EmptyAccessToken:   errEmptySpotifyAccessToken,
		Fetch:              a.fetchAccessToken,
		RefreshTimeout:     spotifyTokenRefreshTimeout,
	})
}

func (a *Adapter) hasCredentials() bool {
	return a.tokenSource.CredentialsConfigured()
}

func (a *Adapter) fetchAccessToken(ctx context.Context, credentials auth.ClientCredentials) (auth.Token, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/token"
	var token TokenResponse
	//nolint:wrapcheck // HTTP exchange spec supplies token request/status/decode context.
	if err := httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client: a.client,
			Method: http.MethodPost,
			URL:    endpoint,
			Body:   strings.NewReader(form.Encode()),
			Headers: map[string]string{
				"Content-Type":  "application/x-www-form-urlencoded",
				"Authorization": credentials.BasicAuthorization(),
			},
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build token request",
			ExecuteError: "execute token request",
			StatusError:  httpx.StatusError(errUnexpectedSpotifyTokenStatus),
		},
		DecodeError: "decode token response",
	}, &token); err != nil {
		return auth.Token{}, err
	}
	return auth.Token{
		AccessToken: token.AccessToken,
		ExpiresIn:   time.Duration(token.ExpiresIn) * time.Second,
	}, nil
}

func isSpotifyAPIStatus(err error, statusCode int) bool {
	var apiErr *spotifyAPIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == statusCode
}

func parseInitialState(body []byte) (*initialState, error) {
	matches := initialStatePattern.FindSubmatch(body)
	if len(matches) != 2 {
		return nil, errInitialStateScriptNotFound
	}

	decoded, err := base64.StdEncoding.DecodeString(string(matches[1]))
	if err != nil {
		return nil, fmt.Errorf("decode initial state: %w", errors.Join(errMalformedSpotifyBootstrapState, err))
	}

	var state initialState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, fmt.Errorf("unmarshal initial state: %w", errors.Join(errMalformedSpotifyBootstrapState, err))
	}
	return &state, nil
}
