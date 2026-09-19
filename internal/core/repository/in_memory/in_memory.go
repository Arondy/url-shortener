package in_memory

import (
	"context"
	"sync"
	"time"

	"github.com/Arondy/url-shortener/internal/core/domain"
)

type entry struct {
	originalURL string
	CreatedAt   time.Time
}

type URLShortenerRepository struct {
	urlsTTL        time.Duration
	original2short map[string]string
	short2original map[string]entry
	mu             sync.Mutex
}

func NewURLShortenerRepository(urlsTTL time.Duration) *URLShortenerRepository {
	return &URLShortenerRepository{
		urlsTTL:        urlsTTL,
		original2short: make(map[string]string),
		short2original: make(map[string]entry),
	}
}

func (r *URLShortenerRepository) Get(ctx context.Context, shortCode string) (domain.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, ok := r.short2original[shortCode]
	if !ok {
		return domain.URL{}, domain.ErrShortURLCodeNotFound
	}

	if r.isExpired(entry) {
		delete(r.short2original, shortCode)
		delete(r.original2short, entry.originalURL)
		return domain.URL{}, domain.ErrShortURLCodeExpired
	}

	return domain.URL{
		OriginalURL:  entry.originalURL,
		ShortURLCode: shortCode,
	}, nil
}

func (r *URLShortenerRepository) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	shortCode, originalURLExists := r.original2short[url.OriginalURL]
	oldEntry, shortCodeExists := r.short2original[url.ShortURLCode]

	if originalURLExists {
		if old, ok := r.short2original[shortCode]; ok && !r.isExpired(old) {
			return domain.URL{
				OriginalURL:  url.OriginalURL,
				ShortURLCode: shortCode,
			}, nil
		}
	}
	if shortCodeExists && !r.isExpired(oldEntry) {
		return domain.URL{}, domain.ErrShortURLCodeCollision
	}

	// если обновляем данные, то нашим "новым" url и code
	// могут соответствовать истекшие старые данные
	// тогда их нужно очистить, чтобы не осталось орфанов
	if shortCodeExists {
		delete(r.original2short, oldEntry.originalURL)
	}
	if originalURLExists {
		delete(r.short2original, shortCode)
	}

	r.original2short[url.OriginalURL] = url.ShortURLCode
	r.short2original[url.ShortURLCode] = entry{
		originalURL: url.OriginalURL,
		CreatedAt:   time.Now(),
	}

	return domain.URL{
		OriginalURL:  url.OriginalURL,
		ShortURLCode: url.ShortURLCode,
	}, nil
}

func (r *URLShortenerRepository) isExpired(e entry) bool {
	return time.Since(e.CreatedAt) > r.urlsTTL
}
