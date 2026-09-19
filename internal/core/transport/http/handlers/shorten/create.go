package shorten

import (
	"net/http"

	"github.com/Arondy/url-shortener/internal/config"
	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/Arondy/url-shortener/internal/core/transport/http/handlers"
)

type CreateShortCodeRequest struct {
	OriginalURL string `json:"original_url" validate:"required,min=4,max=1024,url"`
}

func (r CreateShortCodeRequest) ToDomain() domain.URL {
	return domain.URL{
		OriginalURL: r.OriginalURL,
	}
}

func (h *ShortenHandler) Create(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())
	var shortCodeReq CreateShortCodeRequest

	if !handlers.DecodeJSONBody(w, r, logger, &shortCodeReq) {
		return
	}

	if !handlers.ValidateRequest(w, logger, shortCodeReq) {
		return
	}

	url := shortCodeReq.ToDomain()
	createdURL, err := h.shortenerService.Create(r.Context(), url)
	if err != nil {
		logger.Errorw("failed to create short code for url", "error", err, "url", shortCodeReq.OriginalURL)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	resp := h.ShortenResponseFromDomain(createdURL)
	handlers.WriteJSON(w, logger, http.StatusCreated, resp)
}
