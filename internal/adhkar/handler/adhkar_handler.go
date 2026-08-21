package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ilmnafi/backend/internal/adhkar/service"
	"github.com/ilmnafi/backend/internal/middleware"
	"github.com/ilmnafi/backend/internal/response"
)

type AdhkarHandler struct {
	service service.AdhkarService
}

func NewAdhkarHandler(s service.AdhkarService) *AdhkarHandler {
	return &AdhkarHandler{service: s}
}

func (h *AdhkarHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.service.GetCategories(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, cats)
}

func (h *AdhkarHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
		return
	}
	c, err := h.service.GetCategory(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, c)
}

func (h *AdhkarHandler) GetAdhkarByCategory(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
		return
	}
	items, err := h.service.GetAdhkarByCategory(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, items)
}

func (h *AdhkarHandler) GetAdhkarBySlug(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	items, err := h.service.GetAdhkarBySlug(r.Context(), slug)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, items)
}

func (h *AdhkarHandler) GetDhikr(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_DHIKR_ID", "Invalid dhikr id")
		return
	}
	d, sources, err := h.service.GetDhikrWithSources(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]interface{}{"dhikr": d, "sources": sources})
}

func (h *AdhkarHandler) RecordCompletion(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_DHIKR_ID", "Invalid dhikr id")
		return
	}
	var req struct {
		Increment int `json:"increment"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Increment <= 0 {
		req.Increment = 1
	}
	c, err := h.service.RecordCompletion(r.Context(), userID.String(), id, req.Increment)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, c)
}

func (h *AdhkarHandler) GetDailyProgress(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	date := r.URL.Query().Get("date")
	p, err := h.service.GetDailyProgress(r.Context(), userID.String(), date)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, p)
}

func (h *AdhkarHandler) GetProgressHistory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	history, err := h.service.GetCompletionHistory(r.Context(), userID.String(), days)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, history)
}

func (h *AdhkarHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	var req struct {
		CategoryID       int    `json:"category_id"`
		Enabled          bool   `json:"enabled"`
		Time             string `json:"time"`
		NotificationType string `json:"notification_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
		return
	}
	if err := h.service.CreateReminder(r.Context(), userID.String(), req.CategoryID, req.Enabled, req.Time, req.NotificationType); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteCreated(w, map[string]string{"status": "ok"})
}

func (h *AdhkarHandler) UpdateReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
		return
	}
	var req struct {
		Enabled          bool   `json:"enabled"`
		Time             string `json:"time"`
		NotificationType string `json:"notification_type"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid body")
		return
	}
	if err := h.service.UpdateReminder(r.Context(), userID.String(), id, req.Enabled, req.Time, req.NotificationType); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]string{"status": "ok"})
}

func (h *AdhkarHandler) GetReminders(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	rems, err := h.service.GetReminders(r.Context(), userID.String())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, rems)
}

func (h *AdhkarHandler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r)
	if !ok {
		response.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
		return
	}
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
		return
	}
	if err := h.service.DeleteReminder(r.Context(), userID.String(), id); err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, map[string]string{"status": "deleted"})
}
