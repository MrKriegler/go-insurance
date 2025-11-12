package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type ApplicationHandler struct{}

func NewApplicationHandler() *ApplicationHandler { return &ApplicationHandler{} }

func (h *ApplicationHandler) Mount(r chi.Router) {
	r.Route("/applications", func(r chi.Router) {
		r.Post("/", h.Create)
		r.Get("/{application_id}", h.Get)
		r.Patch("/{application_id}", h.Patch)
		r.Post("/{application_id}:submit", h.Submit)
	})
}

func (h *ApplicationHandler) Create(w http.ResponseWriter, r *http.Request) {}
func (h *ApplicationHandler) Get(w http.ResponseWriter, r *http.Request)    {}
func (h *ApplicationHandler) Patch(w http.ResponseWriter, r *http.Request)  {}
func (h *ApplicationHandler) Submit(w http.ResponseWriter, r *http.Request) {}
