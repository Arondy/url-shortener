package url_shortener

import (
	"context"
	"errors"
	"math/rand/v2"

	"github.com/Arondy/url-shortener/internal/core/domain"
)

type URLShortenerRepo interface {
	Get(ctx context.Context, shortCode string) (domain.URL, error)
	Create(ctx context.Context, url domain.URL) (domain.URL, error)
}

type URLShortenerService struct {
	shortenerRepo URLShortenerRepo
}

func NewService(shortenerRepo URLShortenerRepo) *URLShortenerService {
	return &URLShortenerService{
		shortenerRepo: shortenerRepo,
	}
}

func generateShortCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		index := rand.IntN(len(domain.ShortCodeCharset))
		b[i] = domain.ShortCodeCharset[index]
	}

	return string(b)
}

func (s *URLShortenerService) Get(ctx context.Context, shortCode string) (domain.URL, error) {
	return s.shortenerRepo.Get(ctx, shortCode)
}

func (s *URLShortenerService) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	for range domain.ShortCodeCollisionRetries {
		shortCode := generateShortCode(domain.ShortCodeLength)
		url.ShortURLCode = shortCode
		createdURL, err := s.shortenerRepo.Create(ctx, url)
		if err != nil {
			if errors.Is(err, domain.ErrShortURLCodeCollision) {
				continue
			}
			return domain.URL{}, err
		}
		return createdURL, nil
	}
	return domain.URL{}, domain.ErrShortURLCodeCollision
}
