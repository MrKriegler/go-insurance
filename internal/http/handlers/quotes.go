package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type QuoteHandler struct{}

func NewQuoteHandler() *QuoteHandler { return &QuoteHandler{} }

func (h *QuoteHandler) Mount(r chi.Router) {
	r.Route("/quotes", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{quote_id}", h.Get)
	})
}

func (h *QuoteHandler) Create(w http.ResponseWriter, r *http.Request) {}
func (h *QuoteHandler) Get(w http.ResponseWriter, r *http.Request)    {}
