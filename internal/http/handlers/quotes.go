package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

type QuoteHandler struct {
	Svc  core.QuoteService // does Product->Quote pricing (+ optional persistence)
	Repo core.QuoteRepo    // used for GET by ID
	Log  *slog.Logger
}

func NewQuoteHandler(svc core.QuoteService, repo core.QuoteRepo, log *slog.Logger) *QuoteHandler {
	return &QuoteHandler{Svc: svc, Repo: repo, Log: log}
}

func (h *QuoteHandler) Mount(r chi.Router) {
	r.Route("/quotes", func(r chi.Router) {
		r.Post("/", h.Create)       // POST /quotes  (price + persist)
		r.Get("/{quote_id}", h.Get) // GET  /quotes/{quote_id}
	})
}

// Create prices a quote from input and returns the created quote.
// 201: JSON; 400: bad JSON/validation; 404: product not found; 500: internal error.
func (h *QuoteHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in core.QuoteInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		problem.Write(w, http.StatusBadRequest, "Invalid JSON", "Body could not be decoded.")
		return
	}

	quote, err := h.Svc.Price(r.Context(), in)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to price quote")
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(quote); err != nil {
		h.Log.Error("failed to encode quote", "err", err)
	}
}

// Get retrieves a quote by its ULID.
// 200: JSON; 400: missing ID; 404: not found; 500: internal error.
func (h *QuoteHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "quote_id")
	if id == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Quote ID", "Path parameter quote_id is required.")
		return
	}

	quote, err := h.Repo.Get(r.Context(), id)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to get quote")
		return
	}

	if err := json.NewEncoder(w).Encode(quote); err != nil {
		h.Log.Error("failed to encode quote", "quote_id", id, "err", err)
	}
}
