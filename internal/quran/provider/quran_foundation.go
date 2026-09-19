package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ilmnafi/backend/internal/config"
)

const providerName = "quran_foundation"

type QuranFoundationProvider struct {
	client       *http.Client
	apiBaseURL   string
	oauthBaseURL string
	clientID     string
	clientSecret string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func NewQuranProvider(cfg config.QuranConfig) QuranProvider {
	if cfg.Provider != providerName {
		return unavailableProvider{err: unsupportedProvider(cfg.Provider)}
	}
	if cfg.APIBaseURL == "" {
		if cfg.Environment == "production" {
			cfg.APIBaseURL = "https://apis.quran.foundation"
		} else {
			cfg.APIBaseURL = "https://apis-prelive.quran.foundation"
		}
	}
	if cfg.OAuthBaseURL == "" {
		if cfg.Environment == "production" {
			cfg.OAuthBaseURL = "https://oauth2.quran.foundation"
		} else {
			cfg.OAuthBaseURL = "https://prelive-oauth2.quran.foundation"
		}
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &QuranFoundationProvider{
		client:       &http.Client{Timeout: timeout},
		apiBaseURL:   strings.TrimRight(cfg.APIBaseURL, "/"),
		oauthBaseURL: strings.TrimRight(cfg.OAuthBaseURL, "/"),
		clientID:     cfg.ClientID,
		clientSecret: cfg.ClientSecret,
	}
}

type unavailableProvider struct{ err *ProviderError }

func (p unavailableProvider) ListChapters(context.Context, string) ([]Chapter, error) {
	return nil, p.err
}
func (p unavailableProvider) GetChapter(context.Context, int, string) (*Chapter, error) {
	return nil, p.err
}
func (p unavailableProvider) GetVerses(context.Context, int, VerseQuery) (VersePage, error) {
	return VersePage{}, p.err
}
func (p unavailableProvider) GetVerse(context.Context, string, VerseQuery) (*Verse, error) {
	return nil, p.err
}
func (p unavailableProvider) GetJuz(context.Context, int, int) (*Juz, error) { return nil, p.err }
func (p unavailableProvider) GetPage(context.Context, int, VerseQuery) (VersePage, error) {
	return VersePage{}, p.err
}
func (p unavailableProvider) Search(context.Context, SearchQuery) (SearchResponse, error) {
	return SearchResponse{}, p.err
}
func (p unavailableProvider) ListResources(context.Context, string) ([]Resource, error) {
	return nil, p.err
}

func (p *QuranFoundationProvider) ListChapters(ctx context.Context, language string) ([]Chapter, error) {
	var response struct {
		Chapters []Chapter `json:"chapters"`
	}
	query := url.Values{}
	query.Set("language", defaultLanguage(language))
	if err := p.get(ctx, "/content/api/v4/chapters", query, &response); err != nil {
		return nil, err
	}
	for i := range response.Chapters {
		response.Chapters[i].Provider = providerName
	}
	return response.Chapters, nil
}

func (p *QuranFoundationProvider) GetChapter(ctx context.Context, chapterID int, language string) (*Chapter, error) {
	var response struct {
		Chapter Chapter `json:"chapter"`
	}
	query := url.Values{}
	query.Set("language", defaultLanguage(language))
	if err := p.get(ctx, fmt.Sprintf("/content/api/v4/chapters/%d", chapterID), query, &response); err != nil {
		return nil, err
	}
	response.Chapter.Provider = providerName
	return &response.Chapter, nil
}

func (p *QuranFoundationProvider) GetVerses(ctx context.Context, chapterID int, queryParams VerseQuery) (VersePage, error) {
	var response VersePage
	query := verseQuery(queryParams)
	if err := p.get(ctx, fmt.Sprintf("/content/api/v4/verses/by_chapter/%d", chapterID), query, &response); err != nil {
		return VersePage{}, err
	}
	markVerseProvider(&response)
	return response, nil
}

func (p *QuranFoundationProvider) GetVerse(ctx context.Context, verseKey string, queryParams VerseQuery) (*Verse, error) {
	var response struct {
		Verse Verse `json:"verse"`
	}
	if err := p.get(ctx, "/content/api/v4/verses/by_key/"+url.PathEscape(verseKey), verseQuery(queryParams), &response); err != nil {
		return nil, err
	}
	markVerse(&response.Verse)
	return &response.Verse, nil
}

func (p *QuranFoundationProvider) GetJuz(ctx context.Context, juzID int, mushafID int) (*Juz, error) {
	var response struct {
		Juz Juz `json:"juz"`
	}
	query := url.Values{}
	if mushafID > 0 {
		query.Set("mushaf", strconv.Itoa(mushafID))
	}
	if err := p.get(ctx, fmt.Sprintf("/content/api/v4/juzs/%d", juzID), query, &response); err != nil {
		return nil, err
	}
	response.Juz.Provider = providerName
	return &response.Juz, nil
}

func (p *QuranFoundationProvider) GetPage(ctx context.Context, pageID int, queryParams VerseQuery) (VersePage, error) {
	var response VersePage
	if err := p.get(ctx, fmt.Sprintf("/content/api/v4/verses/by_page/%d", pageID), verseQuery(queryParams), &response); err != nil {
		return VersePage{}, err
	}
	markVerseProvider(&response)
	return response, nil
}

func (p *QuranFoundationProvider) Search(ctx context.Context, params SearchQuery) (SearchResponse, error) {
	var response struct {
		Result     SearchResponse `json:"result"`
		Pagination Pagination     `json:"pagination"`
	}
	query := url.Values{}
	query.Set("mode", "advanced")
	query.Set("query", params.Query)
	query.Set("page", strconv.Itoa(params.Page))
	query.Set("size", strconv.Itoa(params.Size))
	query.Set("highlight", "0")
	if params.TranslationIDs != "" {
		query.Set("translation_ids", params.TranslationIDs)
	}
	if err := p.getSearch(ctx, "/api/v1/search", query, &response); err != nil {
		return SearchResponse{}, err
	}
	response.Result.Pagination = response.Pagination
	return response.Result, nil
}

func (p *QuranFoundationProvider) ListResources(ctx context.Context, resourceType string) ([]Resource, error) {
	var response struct {
		Resources []Resource `json:"resources"`
	}
	if err := p.get(ctx, "/content/api/v4/resources/"+url.PathEscape(resourceType), nil, &response); err != nil {
		return nil, err
	}
	for i := range response.Resources {
		response.Resources[i].ResourceType = resourceType
		response.Resources[i].Provider = providerName
	}
	return response.Resources, nil
}

func (p *QuranFoundationProvider) token(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.accessToken != "" && time.Now().Before(p.tokenExpiry) {
		return p.accessToken, nil
	}
	if p.clientID == "" || p.clientSecret == "" {
		return "", unavailable(fmt.Errorf("Quran Foundation credentials are not configured"))
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("scope", "content search")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.oauthBaseURL+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", unavailable(err)
	}
	req.SetBasicAuth(p.clientID, p.clientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", unavailable(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", statusError(resp.StatusCode)
	}
	var body struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", invalidResponse(err)
	}
	if body.AccessToken == "" {
		return "", invalidResponse(fmt.Errorf("token response did not contain access_token"))
	}
	expires := time.Duration(body.ExpiresIn) * time.Second
	if expires <= 0 {
		expires = time.Hour
	}
	p.accessToken = body.AccessToken
	p.tokenExpiry = time.Now().Add(expires - time.Minute)
	return p.accessToken, nil
}

func (p *QuranFoundationProvider) invalidateToken() {
	p.mu.Lock()
	p.accessToken = ""
	p.tokenExpiry = time.Time{}
	p.mu.Unlock()
}

func (p *QuranFoundationProvider) get(ctx context.Context, endpoint string, query url.Values, out interface{}) error {
	return p.doJSON(ctx, p.apiBaseURL, endpoint, query, out)
}

func (p *QuranFoundationProvider) getSearch(ctx context.Context, endpoint string, query url.Values, out interface{}) error {
	return p.doJSON(ctx, p.apiBaseURL, endpoint, query, out)
}

func (p *QuranFoundationProvider) doJSON(ctx context.Context, baseURL, endpoint string, query url.Values, out interface{}) error {
	for authRetry := 0; authRetry < 2; authRetry++ {
		token, err := p.token(ctx)
		if err != nil {
			return err
		}
		for attempt := 0; attempt < 2; attempt++ {
			u, err := url.Parse(baseURL)
			if err != nil {
				return unavailable(err)
			}
			u.Path = path.Join(u.Path, endpoint)
			u.RawQuery = query.Encode()
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
			if err != nil {
				return unavailable(err)
			}
			req.Header.Set("x-auth-token", token)
			req.Header.Set("x-client-id", p.clientID)
			req.Header.Set("Accept", "application/json")
			resp, err := p.client.Do(req)
			if err != nil {
				return unavailable(err)
			}
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
			resp.Body.Close()
			if readErr != nil {
				return unavailable(readErr)
			}
			if resp.StatusCode == http.StatusUnauthorized && authRetry == 0 {
				p.invalidateToken()
				break
			}
			if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
				if attempt == 0 {
					if err := wait(ctx, time.Second); err != nil {
						return unavailable(err)
					}
					continue
				}
				return statusError(resp.StatusCode)
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return statusError(resp.StatusCode)
			}
			if err := json.Unmarshal(body, out); err != nil {
				return invalidResponse(err)
			}
			return nil
		}
	}
	return statusError(http.StatusUnauthorized)
}

func wait(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func defaultLanguage(language string) string {
	if strings.TrimSpace(language) == "" {
		return "en"
	}
	return language
}

func verseQuery(params VerseQuery) url.Values {
	query := url.Values{}
	query.Set("language", defaultLanguage(params.Language))
	query.Set("words", strconv.FormatBool(params.Words))
	query.Set("fields", "text_uthmani,text_imlaei")
	if params.Translations != "" {
		query.Set("translations", params.Translations)
	}
	if params.Tafsirs != "" {
		query.Set("tafsirs", params.Tafsirs)
	}
	if params.Audio > 0 {
		query.Set("audio", strconv.Itoa(params.Audio))
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.PerPage > 0 {
		query.Set("per_page", strconv.Itoa(params.PerPage))
	}
	return query
}

func markVerseProvider(v *VersePage) {
	for i := range v.Verses {
		markVerse(&v.Verses[i])
	}
}
func markVerse(v *Verse) {
	v.Provider = providerName
	for i := range v.Translations {
		v.Translations[i].Provider = providerName
	}
	for i := range v.Tafsirs {
		v.Tafsirs[i].Provider = providerName
	}
}
