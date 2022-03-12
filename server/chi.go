package server

import "github.com/go-chi/chi/v5"

func NewChiHandler() chi.Router {
	return chi.NewRouter()
}
