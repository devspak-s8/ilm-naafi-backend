package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ilmnafi/backend/internal/middleware"
	"github.com/ilmnafi/backend/internal/quran/service"
	"github.com/ilmnafi/backend/internal/response"
)

type QuranHandler struct {
	service service.QuranService
}

func NewQuranHandler(s service.QuranService) *QuranHandler {
	return &QuranHandler{service: s}
}

func (h *QuranHandler) GetSurahs(w http.ResponseWriter, r *http.Request) {
	surahs, err := h.service.GetSurahs(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, surahs)
}

func (h *QuranHandler) GetSurah(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_SURAH_ID", "Invalid surah id")
		return
	}
	surah, ayahs, err := h.service.GetSurah(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]interface{}{"surah": surah, "ayahs": ayahs})
}

func (h *QuranHandler) GetSurahAyahs(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_SURAH_ID", "Invalid surah id")
		return
	}
	_, ayahs, err := h.service.GetSurah(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, ayahs)
}

func (h *QuranHandler) GetAyah(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_AYAH_ID", "Invalid ayah id")
		return
	}
	ayah, err := h.service.GetAyahByID(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, ayah)
}

func (h *QuranHandler) GetJuz(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_JUZ_ID", "Invalid juz id")
		return
	}
	ayahs, err := h.service.GetJuz(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, ayahs)
}

func (h *QuranHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	results, err := h.service.Search(r.Context(), query, limit)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, results)
}

func (h *QuranHandler) GetReciters(w http.ResponseWriter, r *http.Request) {
	reciters, err := h.service.GetReciters(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, reciters)
}

func (h *QuranHandler) GetReciter(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_RECITER_ID", "Invalid reciter id")
		return
	}
	reciter, err := h.service.GetReciter(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, reciter)
}

func (h *QuranHandler) GetSurahAudio(w http.ResponseWriter, r *http.Request) {
	surahID, err := parseID(mux.Vars(r)["surahID"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_SURAH_ID", "Invalid surah id")
		return
	}
	reciterID, err := parseID(r.URL.Query().Get("reciter"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_RECITER_ID", "Invalid reciter id")
		return
	}
	audio, err := h.service.GetAudio(r.Context(), surahID, reciterID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, audio)
}

func (h *QuranHandler) GetAyahAudio(w http.ResponseWriter, r *http.Request) {
	ayahID, err := parseID(mux.Vars(r)["ayahID"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_AYAH_ID", "Invalid ayah id")
		return
	}
	reciterID, err := parseID(r.URL.Query().Get("reciter"))
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_RECITER_ID", "Invalid reciter id")
		return
	}
	audio, err := h.service.GetAyahAudio(r.Context(), ayahID, reciterID)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, audio)
}

func (h *QuranHandler) GetBookmarks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	bookmarks, err := h.service.GetBookmarks(r.Context(), userID.String())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, bookmarks)
}

func (h *QuranHandler) CreateBookmark(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var req struct {
		AyahID int `json:"ayah_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.AyahID <= 0 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Valid ayah_id is required")
		return
	}
	if err := h.service.CreateBookmark(r.Context(), userID.String(), req.AyahID); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteCreated(w, map[string]string{"status": "ok"})
}

func (h *QuranHandler) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	ayahID, err := parseID(mux.Vars(r)["ayahID"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_AYAH_ID", "Invalid ayah id")
		return
	}
	if err := h.service.DeleteBookmark(r.Context(), userID.String(), ayahID); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]string{"status": "deleted"})
}

func (h *QuranHandler) GetProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	progress, err := h.service.GetProgress(r.Context(), userID.String())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	if progress == nil {
		response.WriteSuccess(w, map[string]interface{}{})
		return
	}
	response.WriteSuccess(w, progress)
}

func (h *QuranHandler) UpdateProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var req struct {
		SurahID    int  `json:"surah_id"`
		AyahID     int  `json:"ayah_id"`
		PageNumber *int `json:"page_number"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.SurahID <= 0 || req.AyahID <= 0 {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Valid surah_id and ayah_id are required")
		return
	}
	if err := h.service.UpdateProgress(r.Context(), userID.String(), req.SurahID, req.AyahID, req.PageNumber); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]string{"status": "ok"})
}

func (h *QuranHandler) ContinueReading(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	progress, err := h.service.ContinueReading(r.Context(), userID.String())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	if progress == nil {
		response.WriteSuccess(w, map[string]interface{}{})
		return
	}
	response.WriteSuccess(w, progress)
}

func parseID(s string) (int, error) {
	return strconv.Atoi(s)
}
