package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type UWHandler struct{}

func NewUWHandler() *UWHandler { return &UWHandler{} }

func (h *UWHandler) Mount(r chi.Router) {
	r.Route("/underwriting", func(r chi.Router) {
		r.Get("/cases/{case_id}", h.GetCase)
	})
}

func (h *UWHandler) GetCase(w http.ResponseWriter, r *http.Request) {}
