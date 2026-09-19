package shorten

import (
	"context"
	"strings"

	"github.com/Arondy/url-shortener/internal/core/domain"
)

type URLShortenerService interface {
	Get(ctx context.Context, shortCode string) (domain.URL, error)
	Create(ctx context.Context, url domain.URL) (domain.URL, error)
}

type ShortenHandler struct {
	shortenerService URLShortenerService
	domainURL        string
}

func NewShortenHandler(domainURL string, shortenerService URLShortenerService) *ShortenHandler {
	return &ShortenHandler{
		shortenerService: shortenerService,
		domainURL:        domainURL,
	}
}

type ShortenResponse struct {
	OriginalURL string `json:"original_url"`
	ShortURL    string `json:"short_url"`
}

func (h *ShortenHandler) ShortenResponseFromDomain(u domain.URL) ShortenResponse {
	return ShortenResponse{
		OriginalURL: u.OriginalURL,
		ShortURL:    strings.TrimRight(h.domainURL, "/") + "/" + u.ShortURLCode,
	}
}
