package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

type ApplicationHandler struct {
	Svc core.ApplicationService
	Log *slog.Logger
}

func NewApplicationHandler(svc core.ApplicationService, log *slog.Logger) *ApplicationHandler {
	return &ApplicationHandler{Svc: svc, Log: log}
}

func (h *ApplicationHandler) Mount(r chi.Router) {
	r.Route("/applications", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{application_id}", h.Get)
		r.Patch("/{application_id}", h.Patch)
		r.Post("/{application_id}:submit", h.Submit)
	})
}

// Create creates a new application from a quote.
// 201: JSON; 400: bad JSON/validation; 404: quote not found; 409: quote already used; 500: internal error.
func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in core.ApplicationInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		problem.Write(w, http.StatusBadRequest, "Invalid JSON", "Body could not be decoded.")
		return
	}

	app, err := h.Svc.Create(r.Context(), in)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(app); err != nil {
		h.Log.Error("failed to encode application", "err", err)
	}
}

// Get retrieves an application by ID.
// 200: JSON; 400: missing ID; 404: not found; 500: internal error.
func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "application_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Application ID", "Path parameter application_id is required.")
		return
	}

	app, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to get application")
		return
	}

	if err := json.NewEncoder(w).Encode(app); err != nil {
		h.Log.Error("failed to encode application", "application_id", id, "err", err)
	}
}

// Patch updates an application (only in draft status).
// 200: JSON; 400: bad JSON/validation; 404: not found; 409: not in draft status; 500: internal error.
func (h *ApplicationHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "application_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Application ID", "Path parameter application_id is required.")
		return
	}

	var patch core.ApplicationPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		problem.Write(w, http.StatusBadRequest, "Invalid JSON", "Body could not be decoded.")
		return
	}

	app, err := h.Svc.Patch(r.Context(), id, patch)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	if err := json.NewEncoder(w).Encode(app); err != nil {
		h.Log.Error("failed to encode application", "application_id", id, "err", err)
	}
}

// Submit submits an application for underwriting.
// 200: JSON; 400: incomplete application; 404: not found; 409: already submitted; 500: internal error.
func (h *ApplicationHandler) Submit(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "application_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Application ID", "Path parameter application_id is required.")
		return
	}

	app, err := h.Svc.Submit(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	if err := json.NewEncoder(w).Encode(app); err != nil {
		h.Log.Error("failed to encode application", "application_id", id, "err", err)
	}
}
