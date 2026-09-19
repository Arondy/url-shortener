package core_http

import "net/http"

type ShortenAPI interface {
	Get(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
}
