package core_http

import (
	"net/http"
)

func NewRouter(shorten ShortenAPI) *http.ServeMux {
	mux := http.NewServeMux()
	registerRoutes(mux, shorten)
	return mux
}

func registerRoutes(mux *http.ServeMux, shorten ShortenAPI) {
	mux.HandleFunc("POST /api/v1/shorten", shorten.Create)
	mux.HandleFunc("GET /api/v1/shorten/{code}", shorten.Get)
}
