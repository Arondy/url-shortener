package in_memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/Arondy/url-shortener/internal/core/repository/in_memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAndGet_RoundTrip(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository()
	ctx := context.Background()

	created, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "abc123_XYZ"})
	require.NoError(t, err)
	assert.Equal(t, "abc123_XYZ", created.ShortURLCode)

	got, err := r.Get(ctx, "abc123_XYZ")
	require.NoError(t, err)
	assert.Equal(t, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "abc123_XYZ"}, got)
}

func TestCreate_SameOriginalReturnsSameCode(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository()
	ctx := context.Background()

	first, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "code111111"})
	require.NoError(t, err)

	second, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "code222222"})
	require.NoError(t, err)
	assert.Equal(t, first.ShortURLCode, second.ShortURLCode)
	assert.Equal(t, "code111111", second.ShortURLCode)
}

func TestCreate_CollisionOnShortCode(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository()
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "samecode12"})
	require.NoError(t, err)

	_, err = r.Create(ctx, domain.URL{OriginalURL: "https://b.com", ShortURLCode: "samecode12"})
	require.ErrorIs(t, err, domain.ErrShortURLCodeCollision)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository()

	_, err := r.Get(context.Background(), "missing123")
	require.ErrorIs(t, err, domain.ErrShortURLCodeNotFound)
}

func TestConcurrent_CreateAndGet(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository()
	ctx := context.Background()

	const n = 100
	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := range n {
		wg.Go(func() {
			code := fmt.Sprintf("code%06d_", i)

			if _, err := r.Create(ctx, domain.URL{
				OriginalURL:  fmt.Sprintf("https://example.com/%d", i),
				ShortURLCode: code,
			}); err != nil {
				errs[i] = err
				return
			}
			if _, err := r.Get(ctx, code); err != nil {
				errs[i] = err
			}
		})
	}

	wg.Wait()
	for _, err := range errs {
		assert.NoError(t, err)
	}
}
