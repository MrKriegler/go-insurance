package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ProductHandler struct{}

func NewProductHandler() *ProductHandler { return &ProductHandler{} }

func (h *ProductHandler) Mount(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{product_id}", h.Get)
	})
}

func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {}
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request)  {}
