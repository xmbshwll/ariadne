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

	"github.com/xmbshwll/ariadne/internal/adapters/adapterutil"
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

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	token, err := a.accessToken(ctx)
	if err != nil {
		return err
	}

	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return adapterutil.GetJSON(ctx, adapterutil.JSONRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client:       a.client,
			URL:          endpoint,
			Headers:      map[string]string{"Authorization": "Bearer " + token},
			UserAgent:    adapterutil.DefaultUserAgent,
			BuildError:   "build api request",
			ExecuteError: "execute api request",
			StatusError: func(statusCode int, body string) error {
				return &spotifyAPIError{StatusCode: statusCode, Message: body}
			},
		},
		DecodeError:       "decode api response",
		MalformedResponse: errMalformedSpotifyAPIResponse,
	}, target)
}

func (a *Adapter) accessToken(ctx context.Context) (string, error) {
	if !a.hasCredentials() {
		return "", ErrCredentialsNotConfigured
	}

	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()

	if a.token.AccessToken != "" && time.Now().Before(a.token.ExpiresAt) {
		return a.token.AccessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/token"
	var token tokenResponse
	//nolint:wrapcheck // HTTP exchange spec supplies token request/status/decode context.
	if err := adapterutil.GetJSON(ctx, adapterutil.JSONRequest{
		RequestSpec: adapterutil.RequestSpec{
			Client: a.client,
			Method: http.MethodPost,
			URL:    endpoint,
			Body:   strings.NewReader(form.Encode()),
			Headers: map[string]string{
				"Content-Type":  "application/x-www-form-urlencoded",
				"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(a.clientID+":"+a.clientSecret)),
			},
			UserAgent:    adapterutil.DefaultUserAgent,
			BuildError:   "build token request",
			ExecuteError: "execute token request",
			StatusError:  adapterutil.StatusError(errUnexpectedSpotifyTokenStatus),
		},
		DecodeError: "decode token response",
	}, &token); err != nil {
		return "", err
	}
	if token.AccessToken == "" {
		return "", errEmptySpotifyAccessToken
	}

	ttl := max(token.ExpiresIn-30, 0)
	a.token = cachedToken{
		AccessToken: token.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(ttl) * time.Second),
	}
	return a.token.AccessToken, nil
}

func (a *Adapter) hasCredentials() bool {
	return strings.TrimSpace(a.clientID) != "" && strings.TrimSpace(a.clientSecret) != ""
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
