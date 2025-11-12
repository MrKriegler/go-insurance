package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type PolicyHandler struct{}

func NewPolicyHandler() *PolicyHandler { return &PolicyHandler{} }

func (h *PolicyHandler) Mount(r chi.Router) {
	r.Route("/policies", func(r chi.Router) {
		r.Get("/{policy_number}", h.Get)
		r.Get("/", h.List) // e.g. ?application_id=...&page[size]=..&page[token]=..
	})
}

func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request)  {}
func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {}
