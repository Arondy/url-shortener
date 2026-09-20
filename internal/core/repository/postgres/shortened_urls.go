package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Arondy/url-shortener/internal/core/domain"
	"github.com/jackc/pgx/v5"
)

type URLShortenerRepository struct {
	*DB
}

func NewURLShortenerRepository(db *DB) *URLShortenerRepository {
	return &URLShortenerRepository{
		DB: db,
	}
}

func (r *URLShortenerRepository) Get(ctx context.Context, shortCode string) (domain.URL, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
	SELECT original_url, created_at
	FROM shortened_urls
	WHERE short_url_code = $1
	`

	row := r.pool.QueryRow(reqCtx, query, shortCode)
	url := domain.URL{
		ShortURLCode: shortCode,
	}

	err := row.Scan(
		&url.OriginalURL,
		&url.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.URL{}, domain.ErrShortURLCodeNotFound
		}
		return domain.URL{}, fmt.Errorf("failed to get original url for '%s' code: %w", shortCode, err)
	}

	if !r.isExpired(url.CreatedAt) {
		return url, nil
	}

	query = `
	DELETE FROM shortened_urls
	WHERE short_url_code = $1 AND created_at < $2
	`

	_, err = r.pool.Exec(reqCtx, query, shortCode, time.Now().Add(-r.urlsTTL))
	if err != nil {
		return domain.URL{}, fmt.Errorf("failed to delete row with '%s' code: %w", shortCode, err)
	}

	return domain.URL{}, domain.ErrShortURLCodeExpired
}

func (r *URLShortenerRepository) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	old, err := r.checkOriginal(reqCtx, url.OriginalURL)
	if err != nil {
		return domain.URL{}, err
	}
	if old != nil {
		return *old, nil
	}

	err = r.checkCollision(reqCtx, url.ShortURLCode)
	if err != nil {
		return domain.URL{}, err
	}

	return r.insertNew(reqCtx, url)
}

func (r *URLShortenerRepository) checkOriginal(ctx context.Context, originalURL string) (*domain.URL, error) {
	query := `
    SELECT original_url, short_url_code, created_at
    FROM shortened_urls
    WHERE original_url = $1
	`

	row := r.pool.QueryRow(ctx, query, originalURL)

	var url domain.URL
	err := row.Scan(
		&url.OriginalURL,
		&url.ShortURLCode,
		&url.CreatedAt,
	)
	if err == nil {
		if !r.isExpired(url.CreatedAt) {
			return &url, nil
		}

		query = `
		DELETE FROM shortened_urls
		WHERE original_url = $1 AND created_at = $2
		`

		_, dErr := r.pool.Exec(ctx, query, url.OriginalURL, url.CreatedAt)
		if dErr != nil {
			return nil, fmt.Errorf("failed to delete row with '%s' original url: %w", originalURL, dErr)
		}
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("failed to scan original: %w", err)
	}

	return nil, nil
}

func (r *URLShortenerRepository) checkCollision(ctx context.Context, shortCode string) error {
	query := `
	SELECT original_url, short_url_code, created_at
	FROM shortened_urls
	WHERE short_url_code = $1
	`

	row := r.pool.QueryRow(ctx, query, shortCode)

	var url domain.URL
	err := row.Scan(
		&url.OriginalURL,
		&url.ShortURLCode,
		&url.CreatedAt,
	)
	if err == nil {
		if !r.isExpired(url.CreatedAt) {
			return domain.ErrShortURLCodeCollision
		}

		query = `
		DELETE FROM shortened_urls
		WHERE short_url_code = $1 AND created_at = $2
		`

		_, dErr := r.pool.Exec(ctx, query, url.ShortURLCode, url.CreatedAt)
		if dErr != nil {
			return fmt.Errorf("failed to delete row with '%s' short code: %w", url.ShortURLCode, dErr)
		}
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to scan shortened: %w", err)
	}
	return nil
}

func (r *URLShortenerRepository) insertNew(ctx context.Context, url domain.URL) (domain.URL, error) {
	query := `
	INSERT INTO shortened_urls (original_url, short_url_code)
	VALUES ($1, $2)
	ON CONFLICT DO NOTHING
	RETURNING original_url, short_url_code, created_at
	`

	row := r.pool.QueryRow(ctx, query, url.OriginalURL, url.ShortURLCode)

	var createdURL domain.URL
	err := row.Scan(
		&createdURL.OriginalURL,
		&createdURL.ShortURLCode,
		&createdURL.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		query = `
		SELECT original_url, short_url_code, created_at
		FROM shortened_urls
		WHERE original_url = $1
		`

		row = r.pool.QueryRow(ctx, query, url.OriginalURL)
		err = row.Scan(
			&createdURL.OriginalURL,
			&createdURL.ShortURLCode,
			&createdURL.CreatedAt,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.URL{}, domain.ErrShortURLCodeCollision
		}
	}
	if err != nil {
		return domain.URL{}, fmt.Errorf("failed to insert url: %w", err)
	}

	return createdURL, nil
}

func (r *URLShortenerRepository) ClearExpired(ctx context.Context) {
	ticker := time.NewTicker(r.clearFrequency)
	defer ticker.Stop()

	for {
		reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)

		query := `
		DELETE FROM shortened_urls
		WHERE created_at < $1
		`

		r.pool.Exec(reqCtx, query, time.Now().Add(-r.urlsTTL))
		cancel()

		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			return
		}
	}
}

func (r *URLShortenerRepository) isExpired(createdAt time.Time) bool {
	return time.Since(createdAt) > r.urlsTTL
}
