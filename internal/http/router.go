package transporthttp

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/MrKriegler/go-insurance/internal/http/handlers"
	"github.com/MrKriegler/go-insurance/internal/middleware"
)

// Deps bundles feature handlers that implement handlers.Mountable.
type Deps struct {
	Mounts []handlers.Mountable
}

func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.SetJSONContentType)

	// Mount each feature's routes into this router.
	for _, m := range d.Mounts {
		m.Mount(r)
	}

	return r
}
