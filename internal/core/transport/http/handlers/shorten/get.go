package shorten

import (
	"errors"
	"net/http"

	"github.com/Arondy/url-shortener/internal/config"
	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/Arondy/url-shortener/internal/core/transport/http/handlers"
)

var allowed = make(map[rune]struct{}, len(domain.ShortCodeCharset))

func init() {
	for _, c := range domain.ShortCodeCharset {
		allowed[c] = struct{}{}
	}
}

func validateShortCode(code string) bool {
	if len(code) != domain.ShortCodeLength {
		return false
	}

	for _, char := range code {
		if _, ok := allowed[char]; !ok {
			return false
		}
	}

	return true
}

func (h *ShortenHandler) Get(w http.ResponseWriter, r *http.Request) {
	logger := config.LoggerFromContext(r.Context())

	shortCode := r.PathValue("code")
	if !validateShortCode(shortCode) {
		logger.Warnw("incorrect code", "code", shortCode)
		handlers.WriteError(w, logger, http.StatusBadRequest, "incorrect code in path")
		return
	}

	url, err := h.shortenerService.Get(r.Context(), shortCode)
	if errors.Is(err, domain.ErrShortURLCodeNotFound) {
		logger.Debugw("url with such short code not found", "code", shortCode)
		handlers.WriteError(w, logger, http.StatusNotFound, domain.ErrShortURLCodeNotFound.Error())
		return
	} else if errors.Is(err, domain.ErrShortURLCodeExpired) {
		logger.Debugw("short code expired", "code", shortCode)
		handlers.WriteError(w, logger, http.StatusGone, domain.ErrShortURLCodeExpired.Error())
		return
	} else if err != nil {
		logger.Errorw("failed to get url by short code", "error", err, "code", shortCode)
		handlers.WriteInternalServerError(w, logger)
		return
	}

	resp := h.ShortenResponseFromDomain(url)
	handlers.WriteJSON(w, logger, http.StatusOK, resp)
}
