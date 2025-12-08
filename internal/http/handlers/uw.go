package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

type UWHandler struct {
	Svc core.UnderwritingService
	Log *slog.Logger
}

func NewUWHandler(svc core.UnderwritingService, log *slog.Logger) *UWHandler {
	return &UWHandler{Svc: svc, Log: log}
}

func (h *UWHandler) Mount(r chi.Router) {
	r.Route("/underwriting", func(r chi.Router) {
		r.Get("/cases/{case_id}", h.GetCase)
		r.Get("/cases", h.ListReferred)
		r.Post("/cases/{case_id}:decide", h.Decide)
	})
}

// GetCase retrieves an underwriting case by ID.
// 200: JSON; 400: missing ID; 404: not found; 500: internal error.
func (h *UWHandler) GetCase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "case_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Case ID", "Path parameter case_id is required.")
		return
	}

	uwCase, err := h.Svc.GetCase(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to get underwriting case")
		return
	}

	if err := json.NewEncoder(w).Encode(uwCase); err != nil {
		h.Log.Error("failed to encode uw case", "case_id", id, "err", err)
	}
}

// ListReferred returns cases awaiting manual review.
// 200: JSON array; 500: internal error.
func (h *UWHandler) ListReferred(w http.ResponseWriter, r *http.Request) {
	cases, err := h.Svc.ListReferred(r.Context(), 50)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to list referred cases")
		return
	}

	// Return empty array instead of null
	if cases == nil {
		cases = []core.UnderwritingCase{}
	}

	if err := json.NewEncoder(w).Encode(cases); err != nil {
		h.Log.Error("failed to encode uw cases", "err", err)
	}
}

// Decide makes a manual underwriting decision.
// 200: JSON; 400: bad JSON/validation; 404: not found; 409: already decided; 500: internal error.
func (h *UWHandler) Decide(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "case_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Case ID", "Path parameter case_id is required.")
		return
	}

	var input core.UWDecisionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		problem.Write(w, http.StatusBadRequest, "Invalid JSON", "Body could not be decoded.")
		return
	}

	uwCase, err := h.Svc.MakeDecision(r.Context(), id, input)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	if err := json.NewEncoder(w).Encode(uwCase); err != nil {
		h.Log.Error("failed to encode uw case", "case_id", id, "err", err)
	}
}
