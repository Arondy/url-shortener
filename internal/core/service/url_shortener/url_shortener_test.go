package url_shortener_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/Arondy/url-shortener/internal/core/service/url_shortener"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubRepo struct {
	createFn func(ctx context.Context, url domain.URL) (domain.URL, error)
	getFn    func(ctx context.Context, shortCode string) (domain.URL, error)
	calls    int
	lastCode string
}

func (s *stubRepo) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	s.calls++
	s.lastCode = url.ShortURLCode
	return s.createFn(ctx, url)
}

func (s *stubRepo) Get(ctx context.Context, shortCode string) (domain.URL, error) {
	return s.getFn(ctx, shortCode)
}

func assertValidShortCode(t *testing.T, code string) {
	t.Helper()
	assert.Len(t, code, domain.ShortCodeLength)
	for _, c := range code {
		assert.True(t, strings.ContainsRune(domain.ShortCodeCharset, c), "unexpected rune %q", c)
	}
}

func TestCreate_SuccessFirstTry(t *testing.T) {
	t.Parallel()
	stub := &stubRepo{
		createFn: func(_ context.Context, u domain.URL) (domain.URL, error) {
			return u, nil
		},
	}
	svc := url_shortener.NewService(stub)

	got, err := svc.Create(context.Background(), domain.URL{OriginalURL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got.OriginalURL)
	assertValidShortCode(t, got.ShortURLCode)
	assert.Equal(t, 1, stub.calls)
}

func TestCreate_RetriesOnCollisionThenSucceeds(t *testing.T) {
	t.Parallel()
	stub := &stubRepo{}
	stub.createFn = func(_ context.Context, u domain.URL) (domain.URL, error) {
		if stub.calls == 1 {
			return domain.URL{}, domain.ErrShortURLCodeCollision
		}
		return u, nil
	}
	svc := url_shortener.NewService(stub)

	got, err := svc.Create(context.Background(), domain.URL{OriginalURL: "https://example.com"})
	require.NoError(t, err)
	assertValidShortCode(t, got.ShortURLCode)
	assert.Equal(t, 2, stub.calls)
}

func TestCreate_ExhaustsRetriesOnPersistentCollision(t *testing.T) {
	t.Parallel()
	stub := &stubRepo{
		createFn: func(_ context.Context, _ domain.URL) (domain.URL, error) {
			return domain.URL{}, domain.ErrShortURLCodeCollision
		},
	}
	svc := url_shortener.NewService(stub)

	_, err := svc.Create(context.Background(), domain.URL{OriginalURL: "https://example.com"})
	require.ErrorIs(t, err, domain.ErrShortURLCodeCollision)
	assert.Equal(t, domain.ShortCodeCollisionRetries, stub.calls)
}

func TestCreate_PropagatesNonCollisionError(t *testing.T) {
	t.Parallel()
	sentinel := errors.New("db unavailable")
	stub := &stubRepo{
		createFn: func(_ context.Context, _ domain.URL) (domain.URL, error) {
			return domain.URL{}, sentinel
		},
	}
	svc := url_shortener.NewService(stub)

	_, err := svc.Create(context.Background(), domain.URL{OriginalURL: "https://example.com"})
	require.ErrorIs(t, err, sentinel)
	assert.Equal(t, 1, stub.calls)
}

func TestGet_Success(t *testing.T) {
	t.Parallel()
	want := domain.URL{OriginalURL: "https://example.com", ShortURLCode: "abcdefghij"}
	stub := &stubRepo{
		getFn: func(_ context.Context, shortCode string) (domain.URL, error) {
			assert.Equal(t, "abcdefghij", shortCode)
			return want, nil
		},
	}
	svc := url_shortener.NewService(stub)

	got, err := svc.Get(context.Background(), "abcdefghij")
	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	stub := &stubRepo{
		getFn: func(_ context.Context, _ string) (domain.URL, error) {
			return domain.URL{}, domain.ErrShortURLCodeNotFound
		},
	}
	svc := url_shortener.NewService(stub)

	_, err := svc.Get(context.Background(), "missing123")
	require.ErrorIs(t, err, domain.ErrShortURLCodeNotFound)
}
