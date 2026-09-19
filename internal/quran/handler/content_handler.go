package handler

import (
	"net/http"
	"net/url"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ilmnafi/backend/internal/quran/provider"
	"github.com/ilmnafi/backend/internal/quran/service"
	"github.com/ilmnafi/backend/internal/response"
)

type ContentHandler struct{ service service.ContentService }

func NewContentHandler(svc service.ContentService) *ContentHandler {
	return &ContentHandler{service: svc}
}

func (h *ContentHandler) ListChapters(w http.ResponseWriter, r *http.Request) {
	chapters, err := h.service.ListChapters(r.Context(), r.URL.Query().Get("language"))
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, chapters)
}

func (h *ContentHandler) GetChapter(w http.ResponseWriter, r *http.Request) {
	id, err := positiveParam(mux.Vars(r)["chapter"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CHAPTER", "Invalid chapter")
		return
	}
	chapter, err := h.service.GetChapter(r.Context(), id, r.URL.Query().Get("language"))
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, chapter)
}

func (h *ContentHandler) GetChapterVerses(w http.ResponseWriter, r *http.Request) {
	id, err := positiveParam(mux.Vars(r)["chapter"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CHAPTER", "Invalid chapter")
		return
	}
	verses, err := h.service.GetVerses(r.Context(), id, parseVerseQuery(r))
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, verses)
}

func (h *ContentHandler) GetVerse(w http.ResponseWriter, r *http.Request) {
	key, err := url.PathUnescape(mux.Vars(r)["verseKey"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_VERSE_KEY", "Invalid verse key")
		return
	}
	verse, err := h.service.GetVerse(r.Context(), key, parseVerseQuery(r))
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, verse)
}

func (h *ContentHandler) GetJuz(w http.ResponseWriter, r *http.Request) {
	id, err := positiveParam(mux.Vars(r)["juz"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JUZ", "Invalid juz")
		return
	}
	mushafID, _ := strconv.Atoi(r.URL.Query().Get("mushaf_id"))
	juz, err := h.service.GetJuz(r.Context(), id, mushafID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, juz)
}

func (h *ContentHandler) GetPage(w http.ResponseWriter, r *http.Request) {
	id, err := positiveParam(mux.Vars(r)["page"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_PAGE", "Invalid page")
		return
	}
	page, err := h.service.GetPage(r.Context(), id, parseVerseQuery(r))
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, page)
}

func (h *ContentHandler) Search(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	results, err := h.service.Search(r.Context(), provider.SearchQuery{Query: r.URL.Query().Get("q"), Page: page, Size: size, TranslationIDs: r.URL.Query().Get("translation_ids")})
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, results)
}

func (h *ContentHandler) ListResources(w http.ResponseWriter, r *http.Request) {
	resources, err := h.service.ListResources(r.Context(), mux.Vars(r)["resource"])
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, resources)
}

func positiveParam(value string) (int, error) {
	id, err := strconv.Atoi(value)
	if err != nil || id <= 0 {
		return 0, err
	}
	return id, nil
}

func parseVerseQuery(r *http.Request) provider.VerseQuery {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	audio, _ := strconv.Atoi(r.URL.Query().Get("audio"))
	words, _ := strconv.ParseBool(r.URL.Query().Get("words"))
	return provider.VerseQuery{
		Language:     r.URL.Query().Get("language"),
		Words:        words,
		Translations: r.URL.Query().Get("translations"),
		Tafsirs:      r.URL.Query().Get("tafsirs"),
		Audio:        audio,
		Page:         page,
		PerPage:      perPage,
	}
}
