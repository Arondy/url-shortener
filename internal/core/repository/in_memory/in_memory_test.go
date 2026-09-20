package in_memory_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/Arondy/url-shortener/internal/core/repository/in_memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTTL = time.Hour

func TestCreateAndGet_RoundTrip(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(testTTL)
	ctx := context.Background()

	created, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "abc123_XYZ"})
	require.NoError(t, err)
	assert.Equal(t, "abc123_XYZ", created.ShortURLCode)
	assert.False(t, created.CreatedAt.IsZero())
	assert.WithinDuration(t, time.Now(), created.CreatedAt, 5*time.Second)

	got, err := r.Get(ctx, "abc123_XYZ")
	require.NoError(t, err)
	assert.Equal(t, created.OriginalURL, got.OriginalURL)
	assert.Equal(t, created.ShortURLCode, got.ShortURLCode)
	assert.True(t, created.CreatedAt.Equal(got.CreatedAt), "Get must return CreatedAt from Create")
}

func TestCreate_SameOriginalReturnsSameCode(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(testTTL)
	ctx := context.Background()

	first, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "code111111"})
	require.NoError(t, err)

	second, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "code222222"})
	require.NoError(t, err)
	assert.Equal(t, first.ShortURLCode, second.ShortURLCode)
	assert.Equal(t, "code111111", second.ShortURLCode)
	assert.True(t, first.CreatedAt.Equal(second.CreatedAt), "idempotent Create must return original CreatedAt")
}

func TestCreate_CollisionOnShortCode(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(testTTL)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "samecode12"})
	require.NoError(t, err)

	_, err = r.Create(ctx, domain.URL{OriginalURL: "https://b.com", ShortURLCode: "samecode12"})
	require.ErrorIs(t, err, domain.ErrShortURLCodeCollision)
}

func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(testTTL)

	_, err := r.Get(context.Background(), "missing123")
	require.ErrorIs(t, err, domain.ErrShortURLCodeNotFound)
}

func TestGet_Expired(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(50 * time.Millisecond)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://example.com", ShortURLCode: "expiring01"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	_, err = r.Get(ctx, "expiring01")
	require.ErrorIs(t, err, domain.ErrShortURLCodeExpired)

	// истекшая запись должна удаляться: второй Get NotFound, а не Expired
	_, err = r.Get(ctx, "expiring01")
	require.ErrorIs(t, err, domain.ErrShortURLCodeNotFound)
}

func TestCreate_ExpiredShortCanBeReused(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(50 * time.Millisecond)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "reuse00001"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	created, err := r.Create(ctx, domain.URL{OriginalURL: "https://b.com", ShortURLCode: "reuse00001"})
	require.NoError(t, err)
	assert.Equal(t, "reuse00001", created.ShortURLCode)

	got, err := r.Get(ctx, "reuse00001")
	require.NoError(t, err)
	assert.Equal(t, "https://b.com", got.OriginalURL)

	// висячий обратный маппинг a.com, reuse00001 должен исчезнуть
	recreated, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "fresh00001"})
	require.NoError(t, err)
	assert.Equal(t, "fresh00001", recreated.ShortURLCode)
}

func TestCreate_ExpiredOriginalCanBeRewritten(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(50 * time.Millisecond)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "oldcode001"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	created, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "newcode001"})
	require.NoError(t, err)
	assert.Equal(t, "newcode001", created.ShortURLCode)

	got, err := r.Get(ctx, "newcode001")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", got.OriginalURL)

	// висячий короткий код удаляется при перезаписи
	_, err = r.Get(ctx, "oldcode001")
	require.ErrorIs(t, err, domain.ErrShortURLCodeNotFound)
}

func TestCreate_SamePairAfterExpiry(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(50 * time.Millisecond)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "samepair01"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	created, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "samepair01"})
	require.NoError(t, err)
	assert.Equal(t, "samepair01", created.ShortURLCode)

	got, err := r.Get(ctx, "samepair01")
	require.NoError(t, err)
	assert.Equal(t, "https://a.com", got.OriginalURL)
}

func TestCreate_ExpiredOriginalWithCollidingFreshShort(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(50 * time.Millisecond)
	ctx := context.Background()

	_, err := r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "oldcode001"})
	require.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	// c.com занимает свежий код уже после истечения a.com
	_, err = r.Create(ctx, domain.URL{OriginalURL: "https://c.com", ShortURLCode: "fresh00001"})
	require.NoError(t, err)

	_, err = r.Create(ctx, domain.URL{OriginalURL: "https://a.com", ShortURLCode: "fresh00001"})
	require.ErrorIs(t, err, domain.ErrShortURLCodeCollision)
}

func TestConcurrent_CreateAndGet(t *testing.T) {
	t.Parallel()
	r := in_memory.NewURLShortenerRepository(testTTL)
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
