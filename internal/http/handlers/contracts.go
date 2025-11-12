package handlers

import "github.com/go-chi/chi/v5"

type Mountable interface {
	Mount(r chi.Router)
}
