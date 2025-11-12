package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type OfferHandler struct{}

func NewOfferHandler() *OfferHandler { return &OfferHandler{} }

func (h *OfferHandler) Mount(r chi.Router) {
	// Create under applications; manage via /offers
	r.Post("/applications/{application_id}/offers", h.Create)

	r.Route("/offers", func(r chi.Router) {
		r.Get("/{offer_id}", h.Get)
		r.Post("/{offer_id}:accept", h.Accept)
		r.Post("/{offer_id}:decline", h.Decline)
	})
}

func (h *OfferHandler) Create(w http.ResponseWriter, r *http.Request)  {}
func (h *OfferHandler) Get(w http.ResponseWriter, r *http.Request)     {}
func (h *OfferHandler) Accept(w http.ResponseWriter, r *http.Request)  {}
func (h *OfferHandler) Decline(w http.ResponseWriter, r *http.Request) {}
