package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ilmnafi/backend/internal/config"
	"github.com/ilmnafi/backend/internal/quran/provider"
	"github.com/stretchr/testify/require"
)

func TestQuranFoundationProviderUsesServerCredentialsAndHeaders(t *testing.T) {
	var tokenRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth2/token":
			atomic.AddInt32(&tokenRequests, 1)
			requireBasicAuth(t, r, "client", "secret")
			require.Equal(t, "client_credentials", r.FormValue("grant_type"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "token", "expires_in": 3600})
		case "/content/api/v4/chapters":
			require.Equal(t, "token", r.Header.Get("x-auth-token"))
			require.Equal(t, "client", r.Header.Get("x-client-id"))
			require.Equal(t, "en", r.URL.Query().Get("language"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"chapters": []map[string]interface{}{{"id": 1, "name_simple": "Al-Fatihah", "name_arabic": "الفاتحة", "verses_count": 7}}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	p := provider.NewQuranProvider(config.QuranConfig{
		Provider: "quran_foundation", ClientID: "client", ClientSecret: "secret",
		APIBaseURL: server.URL, OAuthBaseURL: server.URL, Timeout: time.Second,
	})
	chapters, err := p.ListChapters(context.Background(), "")
	require.NoError(t, err)
	require.Len(t, chapters, 1)
	require.Equal(t, "quran_foundation", chapters[0].Provider)
	require.Equal(t, int32(1), atomic.LoadInt32(&tokenRequests))
}

func TestQuranFoundationProviderRenewsTokenOnceAfterUnauthorized(t *testing.T) {
	var tokenRequests int32
	var contentRequests int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			tokenNumber := atomic.AddInt32(&tokenRequests, 1)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "token-" + string(rune('0'+tokenNumber)), "expires_in": 3600})
			return
		}
		if r.URL.Path == "/content/api/v4/chapters" {
			if atomic.AddInt32(&contentRequests, 1) == 1 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			require.Equal(t, "token-2", r.Header.Get("x-auth-token"))
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"chapters": []map[string]interface{}{}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	p := provider.NewQuranProvider(config.QuranConfig{
		Provider: "quran_foundation", ClientID: "client", ClientSecret: "secret",
		APIBaseURL: server.URL, OAuthBaseURL: server.URL, Timeout: time.Second,
	})
	_, err := p.ListChapters(context.Background(), "en")
	require.NoError(t, err)
	require.Equal(t, int32(2), atomic.LoadInt32(&tokenRequests))
	require.Equal(t, int32(2), atomic.LoadInt32(&contentRequests))
}

func TestQuranFoundationProviderMapsNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth2/token" {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"access_token": "token", "expires_in": 3600})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	p := provider.NewQuranProvider(config.QuranConfig{
		Provider: "quran_foundation", ClientID: "client", ClientSecret: "secret",
		APIBaseURL: server.URL, OAuthBaseURL: server.URL, Timeout: time.Second,
	})
	_, err := p.GetVerse(context.Background(), "1:1", provider.VerseQuery{})
	require.Error(t, err)
	providerErr, ok := err.(*provider.ProviderError)
	require.True(t, ok)
	require.Equal(t, "QURAN_RESOURCE_NOT_FOUND", providerErr.Code)
}

func requireBasicAuth(t *testing.T, r *http.Request, expectedUser, expectedPassword string) {
	t.Helper()
	user, password, ok := r.BasicAuth()
	require.True(t, ok)
	require.Equal(t, expectedUser, user)
	require.Equal(t, expectedPassword, password)
	form, _ := url.ParseQuery(readBody(t, r))
	require.True(t, strings.Contains(form.Get("scope"), "content"))
}

func readBody(t *testing.T, r *http.Request) string {
	t.Helper()
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	return r.Form.Encode()
}
