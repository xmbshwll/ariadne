package validation_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xmbshwll/ariadne/cmd/internal/validation"
)

var (
	errTokenStatus  = errors.New("unexpected token status")
	errTokenMissing = errors.New("token response did not include access_token")
	errGetStatus    = errors.New("unexpected api status")
)

// TestFetchClientCredentialsToken pins the shared client-credentials exchange:
// both credential styles authenticate, the token is read from access_token,
// and a tokenless response answers the tool's missing sentinel.
func TestFetchClientCredentialsToken(t *testing.T) {
	tests := []struct {
		name         string
		useBasicAuth bool
		handler      http.HandlerFunc
		wantToken    string
		wantErr      error
	}{
		{
			name:         "form credentials authenticate and return the token",
			useBasicAuth: false,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.NoError(t, r.ParseForm())
				assert.Equal(t, "client_credentials", r.Form.Get("grant_type"))
				assert.Equal(t, "the-id", r.Form.Get("client_id"))
				assert.Equal(t, "the-secret", r.Form.Get("client_secret"))
				assert.NotEqual(t, "Bearer", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"access_token":"token-1"}`))
			},
			wantToken: "token-1",
		},
		{
			name:         "basic auth credentials authenticate and return the token",
			useBasicAuth: true,
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.NoError(t, r.ParseForm())
				assert.NotContains(t, r.Form, "client_id")
				user, pass, ok := r.BasicAuth()
				assert.True(t, ok)
				assert.Equal(t, "the-id", user)
				assert.Equal(t, "the-secret", pass)
				_, _ = w.Write([]byte(`{"access_token":"token-2"}`))
			},
			wantToken: "token-2",
		},
		{
			name:         "a tokenless response answers the missing sentinel",
			useBasicAuth: false,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(`{"access_token":"  "}`))
			},
			wantErr: errTokenMissing,
		},
		{
			name:         "an unexpected status answers the status sentinel",
			useBasicAuth: false,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", http.StatusUnauthorized)
			},
			wantErr: errTokenStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			token, err := validation.FetchClientCredentialsToken(context.Background(), validation.TokenRequest{
				Client:       server.Client(),
				Endpoint:     server.URL + "/token",
				ClientID:     "the-id",
				ClientSecret: "the-secret",
				UseBasicAuth: tt.useBasicAuth,
				BuildError:   "build token request",
				ExecuteError: "execute token request",
				StatusError:  errTokenStatus,
				DecodeError:  "decode token response",
				MissingError: errTokenMissing,
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, tt.name)
				return
			}
			require.NoError(t, err, tt.name)
			assert.Equal(t, tt.wantToken, token, tt.name)
		})
	}
}

// TestAuthenticatedGet pins the shared bearer GET: the token lands in the
// Authorization header, extra headers pass through, and the raw body comes
// back for the tool's own decoding.
func TestAuthenticatedGet(t *testing.T) {
	tests := []struct {
		name     string
		extra    map[string]string
		handler  http.HandlerFunc
		wantBody string
		wantErr  error
	}{
		{
			name: "returns the raw body with the bearer header",
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"ok":true}`))
			},
			wantBody: `{"ok":true}`,
		},
		{
			name:  "extra headers pass through next to the bearer header",
			extra: map[string]string{"Accept": "application/vnd.api+json"},
			handler: func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "application/vnd.api+json", r.Header.Get("Accept"))
				assert.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
				_, _ = w.Write([]byte(`{"ok":true}`))
			},
			wantBody: `{"ok":true}`,
		},
		{
			name: "an unexpected status answers the status sentinel",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "gone", http.StatusNotFound)
			},
			wantErr: errGetStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.handler)
			defer server.Close()

			body, err := validation.AuthenticatedGet(context.Background(), validation.GetRequest{
				Client:       server.Client(),
				URL:          server.URL + "/probe",
				Token:        "token-1",
				Headers:      tt.extra,
				BuildError:   "build api request",
				ExecuteError: "execute api request",
				StatusError:  errGetStatus,
				ReadError:    "read api response",
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr, tt.name)
				return
			}
			require.NoError(t, err, tt.name)
			assert.JSONEq(t, tt.wantBody, string(body), tt.name)
		})
	}
}

// TestDecodeJSONInto pins the labeled decode seam the tools use for payloads.
func TestDecodeJSONInto(t *testing.T) {
	var payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name        string
		raw         string
		wantName    string
		wantErr     error
		wantErrText string
	}{
		{name: "decodes into the target", raw: `{"name":"abbey road"}`, wantName: "abbey road"},
		{name: "answers the labeled decode error", raw: `{broken`, wantErrText: "decode probe payload"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload.Name = ""
			err := validation.DecodeJSONInto([]byte(tt.raw), &payload, "decode probe payload")
			if tt.wantErrText != "" {
				require.ErrorContains(t, err, tt.wantErrText, tt.name)
				return
			}
			require.NoError(t, err, tt.name)
			assert.Equal(t, tt.wantName, payload.Name, tt.name)
		})
	}
}

// TestWriteArtifacts pins the artifact writer: pretty JSON per artifact,
// structured summary last, and empty artifacts skipped.
func TestWriteArtifacts(t *testing.T) {
	tests := []struct {
		name        string
		artifacts   []validation.Artifact
		summary     map[string]any
		wantFiles   []string
		wantSummary bool
	}{
		{
			name: "writes every artifact and the summary",
			artifacts: []validation.Artifact{
				{Name: "source.json", Body: []byte(`{"id":"1"}`)},
				{Name: "search.json", Body: []byte(`{"id":"2"}`)},
			},
			summary:     map[string]any{"album_id": "1"},
			wantFiles:   []string{"source.json", "search.json"},
			wantSummary: true,
		},
		{
			name: "skips empty artifacts but still writes the summary",
			artifacts: []validation.Artifact{
				{Name: "source.json", Body: []byte(`{"id":"1"}`)},
				{Name: "optional.json", Body: nil},
			},
			summary:     map[string]any{"album_id": "1"},
			wantFiles:   []string{"source.json"},
			wantSummary: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputDir := t.TempDir()

			require.NoError(t, validation.WriteArtifacts(outputDir, tt.artifacts, "summary.json", tt.summary), tt.name)

			for _, name := range tt.wantFiles {
				content, err := os.ReadFile(filepath.Join(outputDir, name))
				require.NoError(t, err, tt.name)
				assert.Contains(t, string(content), "{", tt.name)
			}
			summaryPath := filepath.Join(outputDir, "summary.json")
			_, err := os.Stat(summaryPath)
			assert.Equal(t, tt.wantSummary, err == nil, tt.name)
		})
	}
}
