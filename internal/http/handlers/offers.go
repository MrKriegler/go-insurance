package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

type OfferHandler struct {
	Svc core.OfferService
	Log *slog.Logger
}

func NewOfferHandler(svc core.OfferService, log *slog.Logger) *OfferHandler {
	return &OfferHandler{Svc: svc, Log: log}
}

func (h *OfferHandler) Mount(r chi.Router) {
	// Create offer under applications
	r.Post("/applications/{application_id}/offers", h.Create)

	// Manage offers via /offers
	r.Route("/offers", func(r chi.Router) {
		r.Get("/{offer_id}", h.Get)
		r.Post("/{offer_id}:accept", h.Accept)
		r.Post("/{offer_id}:decline", h.Decline)
	})
}

// Create generates an offer from an approved application.
// 201: JSON; 404: application not found; 409: not approved or offer exists; 500: internal error.
func (h *OfferHandler) Create(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "application_id")
	if appID == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Application ID", "Path parameter application_id is required.")
		return
	}

	offer, err := h.Svc.GenerateOffer(r.Context(), appID)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(offer); err != nil {
		h.Log.Error("failed to encode offer", "err", err)
	}
}

// Get retrieves an offer by ID.
// 200: JSON; 400: missing ID; 404: not found; 500: internal error.
func (h *OfferHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "offer_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Offer ID", "Path parameter offer_id is required.")
		return
	}

	offer, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to get offer")
		return
	}

	if err := json.NewEncoder(w).Encode(offer); err != nil {
		h.Log.Error("failed to encode offer", "offer_id", id, "err", err)
	}
}

// Accept accepts an offer.
// 200: JSON; 400: missing ID; 404: not found; 409: expired or not pending; 500: internal error.
func (h *OfferHandler) Accept(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "offer_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Offer ID", "Path parameter offer_id is required.")
		return
	}

	offer, err := h.Svc.Accept(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	if err := json.NewEncoder(w).Encode(offer); err != nil {
		h.Log.Error("failed to encode offer", "offer_id", id, "err", err)
	}
}

// Decline declines an offer.
// 200: JSON; 400: missing ID; 404: not found; 409: not pending; 500: internal error.
func (h *OfferHandler) Decline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "offer_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Offer ID", "Path parameter offer_id is required.")
		return
	}

	offer, err := h.Svc.Decline(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, err.Error())
		return
	}

	if err := json.NewEncoder(w).Encode(offer); err != nil {
		h.Log.Error("failed to encode offer", "offer_id", id, "err", err)
	}
}
