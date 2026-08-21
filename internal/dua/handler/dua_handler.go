package handler

import (
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ilmnafi/backend/internal/dua/service"
	"github.com/ilmnafi/backend/internal/response"
)

type DuaHandler struct {
	service service.DuaService
}

func NewDuaHandler(s service.DuaService) *DuaHandler {
	return &DuaHandler{service: s}
}

func (h *DuaHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := h.service.GetCategories(r.Context())
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, cats)
}

func (h *DuaHandler) GetDuas(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_CATEGORY_ID", "Invalid category id")
		return
	}
	items, err := h.service.GetDuas(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, items)
}

func (h *DuaHandler) GetDua(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, "INVALID_DUA_ID", "Invalid dua id")
		return
	}
	d, err := h.service.GetDua(r.Context(), id)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, d)
}

func (h *DuaHandler) GetDuasBySlug(w http.ResponseWriter, r *http.Request) {
	slug := mux.Vars(r)["slug"]
	items, err := h.service.GetDuasBySlug(r.Context(), slug)
	if err != nil {
		response.WriteAppError(w, err)
		return
	}
	response.WriteSuccess(w, items)
}
