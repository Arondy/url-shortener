package in_memory

import (
	"context"
	"sync"

	"github.com/Arondy/url-shortener/internal/core/domain"
)

type URLShortenerRepository struct {
	original2short map[string]string
	short2original map[string]string
	mu             sync.RWMutex
}

func NewURLShortenerRepository() *URLShortenerRepository {
	return &URLShortenerRepository{
		original2short: make(map[string]string),
		short2original: make(map[string]string),
	}
}

func (r *URLShortenerRepository) Get(ctx context.Context, shortCode string) (domain.URL, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	originalURL, ok := r.short2original[shortCode]

	if !ok {
		return domain.URL{}, domain.ErrShortURLCodeNotFound
	}

	return domain.URL{
		OriginalURL:  originalURL,
		ShortURLCode: shortCode,
	}, nil
}

func (r *URLShortenerRepository) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.short2original[url.ShortURLCode]; ok {
		return domain.URL{}, domain.ErrShortURLCodeCollision
	}

	shortURLCode, ok := r.original2short[url.OriginalURL]
	if !ok {
		r.original2short[url.OriginalURL] = url.ShortURLCode
		r.short2original[url.ShortURLCode] = url.OriginalURL
		shortURLCode = url.ShortURLCode
	}

	return domain.URL{
		OriginalURL:  url.OriginalURL,
		ShortURLCode: shortURLCode,
	}, nil
}
