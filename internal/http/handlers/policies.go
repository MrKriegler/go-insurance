package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/core"
	"github.com/MrKriegler/go-insurance/pkg/problem"
)

type PolicyHandler struct {
	Svc core.PolicyService
	Log *slog.Logger
}

func NewPolicyHandler(svc core.PolicyService, log *slog.Logger) *PolicyHandler {
	return &PolicyHandler{Svc: svc, Log: log}
}

func (h *PolicyHandler) Mount(r chi.Router) {
	r.Route("/policies", func(r chi.Router) {
		r.Get("/{policy_number}", h.Get)
		r.Get("/", h.List)
	})
}

// Get retrieves a policy by its number.
// 200: JSON; 400: missing number; 404: not found; 500: internal error.
func (h *PolicyHandler) Get(w http.ResponseWriter, r *http.Request) {
	number := chi.URLParam(r, "policy_number")
	if number == "" {
		problem.Write(w, http.StatusBadRequest, "Missing Policy Number", "Path parameter policy_number is required.")
		return
	}

	policy, err := h.Svc.GetByNumber(r.Context(), number)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to get policy")
		return
	}

	if err := json.NewEncoder(w).Encode(policy); err != nil {
		h.Log.Error("failed to encode policy", "policy_number", number, "err", err)
	}
}

// List returns policies with optional filtering and pagination.
// 200: JSON; 500: internal error.
func (h *PolicyHandler) List(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := core.PolicyFilter{
		ApplicationID: r.URL.Query().Get("application_id"),
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = core.PolicyStatus(status)
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	policies, total, err := h.Svc.List(r.Context(), filter, limit, offset)
	if err != nil {
		writeError(r.Context(), h.Log, w, err, "Failed to list policies")
		return
	}

	// Return empty array instead of null
	if policies == nil {
		policies = []core.Policy{}
	}

	response := map[string]interface{}{
		"items": policies,
		"total": total,
		"limit": limit,
		"offset": offset,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.Log.Error("failed to encode policies", "err", err)
	}
}
