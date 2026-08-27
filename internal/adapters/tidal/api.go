package tidal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/xmbshwll/ariadne/internal/auth"
	"github.com/xmbshwll/ariadne/internal/httpx"
	"github.com/xmbshwll/ariadne/internal/model"
	"github.com/xmbshwll/ariadne/internal/normalize"
)

const (
	maxTIDALTokenResponseBytes = 16 * 1024
	tidalTokenRefreshTimeout   = 30 * time.Second
)

var errTIDALTokenResponseTooLarge = errors.New("tidal token response too large")

func (a *Adapter) getAPIJSON(ctx context.Context, endpoint string, target any) error {
	token, err := a.AccessToken(ctx)
	if err != nil {
		return err
	}
	//nolint:wrapcheck // HTTP exchange spec supplies request/status/decode context.
	return httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client: a.client,
			URL:    endpoint,
			Headers: map[string]string{
				"Authorization": "Bearer " + token,
				"Accept":        "application/vnd.api+json",
			},
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build api request",
			ExecuteError: "execute api request",
			StatusError:  httpx.StatusError(errUnexpectedTIDALAPIStatus),
		},
		DecodeError:       "decode api response",
		MalformedResponse: ErrMalformedTIDALAPIResponse,
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
		EmptyAccessToken:   errEmptyTIDALAccessToken,
		Fetch:              a.fetchAccessToken,
		RefreshTimeout:     tidalTokenRefreshTimeout,
	})
}

func (a *Adapter) hasCredentials() bool {
	return a.tokenSource.CredentialsConfigured()
}

func (a *Adapter) fetchAccessToken(ctx context.Context, credentials auth.ClientCredentials) (auth.Token, error) {
	form := url.Values{}
	form.Set("client_id", credentials.ClientID)
	form.Set("client_secret", credentials.ClientSecret)
	form.Set("grant_type", "client_credentials")
	endpoint := a.authBaseURL + "/oauth2/token"
	body, err := httpx.FetchBytes(ctx, httpx.BytesRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       a.client,
			Method:       http.MethodPost,
			URL:          endpoint,
			Body:         strings.NewReader(form.Encode()),
			Headers:      map[string]string{"Content-Type": "application/x-www-form-urlencoded; charset=UTF-8"},
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   "build token request",
			ExecuteError: "execute token request",
			StatusError:  httpx.StatusError(errUnexpectedTIDALTokenStatus),
		},
		ReadError:    "read token response",
		MaxBodyBytes: maxTIDALTokenResponseBytes,
		TooLarge: func(maxBytes int64) error {
			return fmt.Errorf("read token response: %w (%d bytes max)", errTIDALTokenResponseTooLarge, maxBytes)
		},
	})
	if err != nil {
		//nolint:wrapcheck // HTTP exchange spec supplies token request/status/read context.
		return auth.Token{}, err
	}
	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return auth.Token{}, fmt.Errorf("decode token response: %w", err)
	}
	return auth.Token{
		AccessToken: token.AccessToken,
		ExpiresIn:   time.Duration(token.ExpiresIn) * time.Second,
	}, nil
}

func (a *Adapter) countryCodeFor(regionHint string) string {
	countryCode := normalizeCountryCode(regionHint)
	if countryCode == "" {
		return normalizeCountryCode(a.defaultCountryCode)
	}
	return countryCode
}

func normalizeCountryCode(value string) string {
	countryCode := strings.ToUpper(strings.TrimSpace(value))
	if len(countryCode) != 2 {
		return ""
	}
	return countryCode
}

func metadataQuery(album model.CanonicalAlbum) string {
	return normalize.SearchPrimaryQuery(album.Title, album.Artists)
}

func songMetadataQuery(song model.CanonicalSong) string {
	return normalize.SearchPrimaryQuery(song.Title, song.Artists)
}

func firstDataResource(document APIDocument) (APIResource, bool, error) {
	resources, err := DocumentData(document)
	if err != nil {
		return APIResource{}, false, err
	}
	if len(resources) == 0 {
		return APIResource{}, false, nil
	}
	return resources[0], true, nil
}

func DocumentData(document APIDocument) ([]APIResource, error) {
	switch data := document.Data.(type) {
	case nil:
		return nil, nil
	case map[string]any:
		content, err := json.Marshal(data)
		if err != nil {
			return nil, ErrMalformedTIDALAPIResponse
		}
		var resource APIResource
		if err := json.Unmarshal(content, &resource); err != nil {
			return nil, ErrMalformedTIDALAPIResponse
		}
		return []APIResource{resource}, nil
	case []any:
		resources := make([]APIResource, 0, len(data))
		for _, item := range data {
			content, err := json.Marshal(item)
			if err != nil {
				return nil, ErrMalformedTIDALAPIResponse
			}
			var resource APIResource
			if err := json.Unmarshal(content, &resource); err != nil {
				return nil, ErrMalformedTIDALAPIResponse
			}
			resources = append(resources, resource)
		}
		return resources, nil
	default:
		return nil, ErrMalformedTIDALAPIResponse
	}
}

// searchResultRelationshipIDs extracts relationship resource IDs from a
// /searchResults document (data holds searchResults resources).
func searchResultRelationshipIDs(document APIDocument, pick func(ResourceRelationships) Relationship) ([]string, error) {
	resources, err := DocumentData(document)
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, resource := range resources {
		if resource.Type != "searchResults" {
			continue
		}
		for _, item := range pick(resource.Relationships).Data {
			if id := strings.TrimSpace(item.ID); id != "" {
				ids = append(ids, id)
			}
		}
	}
	return ids, nil
}

func AlbumIDsFromTrackDocument(document APIDocument) ([]string, error) {
	resources, err := DocumentData(document)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{})
	ids := make([]string, 0, len(resources))
	appendUniqueID := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}

	for _, included := range document.Included {
		if included.Type != "albums" {
			continue
		}
		appendUniqueID(included.ID)
	}
	for _, resource := range resources {
		for _, relation := range resource.Relationships.Albums.Data {
			if relation.Type != "albums" {
				continue
			}
			appendUniqueID(relation.ID)
		}
	}
	return ids, nil
}
