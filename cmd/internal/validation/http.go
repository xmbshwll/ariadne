package validation

// The shared HTTP seam of the validate tools: one client-credentials token
// fetch and one authenticated GET, so a tool states only its endpoints,
// sentinels, and payload shapes. The library's own client-credentials flow is
// internal/auth; these tools need the raw token once, without caching, so the
// exchange lives here next to the runner.

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"strings"

	"github.com/xmbshwll/ariadne/internal/auth"
	"github.com/xmbshwll/ariadne/internal/httpx"
)

// TokenRequest describes one client-credentials token POST.
type TokenRequest struct {
	// Client is the HTTP client the exchange runs on.
	Client *http.Client
	// Endpoint is the full token URL, for example https://auth.tidal.com/v1/oauth2/token.
	Endpoint string
	// ClientID and ClientSecret are the Credential Token inputs.
	ClientID string
	// ClientSecret completes the Credential Token pair.
	ClientSecret string
	// UseBasicAuth sends the credentials as an HTTP Basic header (Spotify)
	// instead of form fields (TIDAL).
	UseBasicAuth bool
	// ContentType overrides the default form content type.
	ContentType string
	// BuildError and ExecuteError label the exchange errors.
	BuildError string
	// ExecuteError completes the exchange error labels.
	ExecuteError string
	// StatusError is the sentinel for an unexpected token status.
	StatusError error
	// DecodeError labels a malformed token response.
	DecodeError string
	// MissingError is the sentinel when the response has no access_token.
	MissingError error
}

// FetchClientCredentialsToken POSTs the client-credentials grant and returns
// the access token, answering MissingError when the response carries none.
func FetchClientCredentialsToken(ctx context.Context, request TokenRequest) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "client_credentials")

	headers := map[string]string{"Content-Type": request.contentType()}
	if request.UseBasicAuth {
		credentials := auth.ClientCredentials{ClientID: request.ClientID, ClientSecret: request.ClientSecret}
		headers["Authorization"] = credentials.BasicAuthorization()
	} else {
		form.Set("client_id", request.ClientID)
		form.Set("client_secret", request.ClientSecret)
	}

	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := httpx.GetJSON(ctx, httpx.JSONRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       request.Client,
			Method:       http.MethodPost,
			URL:          request.Endpoint,
			Body:         strings.NewReader(form.Encode()),
			Headers:      headers,
			UserAgent:    httpx.DefaultUserAgent,
			BuildError:   request.BuildError,
			ExecuteError: request.ExecuteError,
			StatusError:  httpx.StatusError(request.StatusError),
		},
		DecodeError: request.DecodeError,
	}, &token); err != nil {
		return "", err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return "", request.MissingError
	}
	return token.AccessToken, nil
}

func (r TokenRequest) contentType() string {
	if r.ContentType != "" {
		return r.ContentType
	}
	return "application/x-www-form-urlencoded"
}

// GetRequest describes one authenticated GET that keeps the raw body.
type GetRequest struct {
	// Client is the HTTP client the exchange runs on.
	Client *http.Client
	// URL is the full request URL.
	URL string
	// Token is the bearer token for the Authorization header.
	Token string
	// Headers are extra request headers merged under the Authorization header.
	Headers map[string]string
	// UserAgent overrides the default request user agent.
	UserAgent string
	// BuildError and ExecuteError label the exchange errors.
	BuildError string
	// ExecuteError completes the exchange error labels.
	ExecuteError string
	// StatusError is the sentinel for an unexpected response status.
	StatusError error
	// ReadError labels a body read failure.
	ReadError string
}

// AuthenticatedGet performs the bearer-authenticated GET and returns the raw
// response body, so each tool keeps its own payload decoding.
func AuthenticatedGet(ctx context.Context, request GetRequest) ([]byte, error) {
	headers := maps.Clone(request.Headers)
	if headers == nil {
		headers = map[string]string{}
	}
	headers["Authorization"] = "Bearer " + request.Token
	return httpx.FetchBytes(ctx, httpx.BytesRequest{
		RequestSpec: httpx.RequestSpec{
			Client:       request.Client,
			URL:          request.URL,
			Headers:      headers,
			UserAgent:    request.userAgent(),
			BuildError:   request.BuildError,
			ExecuteError: request.ExecuteError,
			StatusError:  httpx.StatusError(request.StatusError),
		},
		ReadError: request.ReadError,
	})
}

func (r GetRequest) userAgent() string {
	if r.UserAgent != "" {
		return r.UserAgent
	}
	return httpx.DefaultUserAgent
}

// Artifact is one named raw JSON body a tool writes next to its summary.
type Artifact struct {
	Name string
	Body []byte
}

// WriteArtifacts writes every artifact as pretty JSON and then the summary as
// structured JSON, both into outputDir.
func WriteArtifacts(outputDir string, artifacts []Artifact, summaryName string, summary map[string]any) error {
	for _, artifact := range artifacts {
		if len(artifact.Body) == 0 {
			continue
		}
		if err := WritePrettyJSON(JoinArtifactPath(outputDir, artifact.Name), artifact.Body); err != nil {
			return err
		}
	}
	if summary != nil {
		if err := WriteJSON(JoinArtifactPath(outputDir, summaryName), summary); err != nil {
			return err
		}
	}
	return nil
}

// JoinArtifactPath joins an output directory and artifact file name with
// forward slashes, which is how summaries reference written files.
func JoinArtifactPath(outputDir string, name string) string {
	return toSlashPaths(outputDir + "/" + name)
}

func toSlashPaths(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

// DecodeJSONInto decodes raw JSON into target with a tool-specific error label.
func DecodeJSONInto(raw []byte, target any, decodeError string) error {
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("%s: %w", decodeError, err)
	}
	return nil
}
