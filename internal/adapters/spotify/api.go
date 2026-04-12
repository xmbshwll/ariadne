package spotify

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("build api request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("execute api request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return &spotifyAPIError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(body))}
	}
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode api response: %w", errors.Join(errMalformedSpotifyAPIResponse, err))
	}
	return nil
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
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(a.clientID+":"+a.clientSecret)))
	req.Header.Set("User-Agent", "ariadne/0.1 (+https://github.com/xmbshwll/ariadne)")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute token request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("%w %d: %s", errUnexpectedSpotifyTokenStatus, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
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
