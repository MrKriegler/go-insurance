package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

// ProductHandler serves /products endpoints.
// Thin layer: request parsing, logging, error mapping, and response encoding.
type ProductHandler struct {
	Repo core.ProductRepo
	Log  *slog.Logger
}

// NewProductHandler wires the repo and logger into the handler.
func NewProductHandler(repo core.ProductRepo, log *slog.Logger) *ProductHandler {
	return &ProductHandler{Repo: repo, Log: log}
}

// Mount registers /products routes under the provided router.
func (h *ProductHandler) Mount(r chi.Router) {
	r.Route("/products", func(r chi.Router) {
		r.Get("/", h.List)              // GET /products
		r.Get("/{product_slug}", h.Get) // GET /products/{product_slug}
	})
}

// List returns all products.
// 200: JSON array; 500: internal error.
func (h *ProductHandler) List(w http.ResponseWriter, r *http.Request) {
	products, err := h.Repo.List(r.Context())
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to list products")
		return
	}

	if err := json.NewEncoder(w).Encode(products); err != nil {
		h.Log.Error("failed to encode products list", "err", err)
	}
}

// Get returns a single product by slug.
// 200: JSON object; 400: missing slug; 404: not found; 500: internal error.
func (h *ProductHandler) Get(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "product_slug")
	if slug == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Product Slug",
			"The URL must include a product_slug path parameter.")
		return
	}

	product, err := h.Repo.GetBySlug(r.Context(), slug)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to retrieve product "+slug)
		return
	}

	if err := json.NewEncoder(w).Encode(product); err != nil {
		h.Log.Error("failed to encode product", "product_slug", slug, "err", err)
	}
}
