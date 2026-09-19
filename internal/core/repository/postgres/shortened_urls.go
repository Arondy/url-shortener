package postgres

import (
	"context"
	"errors"
	"fmt"

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
	SELECT original_url
	FROM shortened_urls
	WHERE short_url_code = $1
	`

	row := r.pool.QueryRow(reqCtx, query, shortCode)
	url := domain.URL{
		ShortURLCode: shortCode,
	}

	err := row.Scan(&url.OriginalURL)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.URL{}, domain.ErrShortURLCodeNotFound
		}
		return domain.URL{}, fmt.Errorf("failed to get original url for %s code: %w", shortCode, err)
	}

	return url, nil
}

func (r *URLShortenerRepository) Create(ctx context.Context, url domain.URL) (domain.URL, error) {
	reqCtx, cancel := context.WithTimeout(ctx, r.requestTimeout)
	defer cancel()

	query := `
    INSERT INTO shortened_urls (original_url, short_url_code)
    VALUES ($1, $2)
    ON CONFLICT DO NOTHING
    RETURNING original_url, short_url_code
	`

	row := r.pool.QueryRow(reqCtx, query, url.OriginalURL, url.ShortURLCode)
	var createdURL domain.URL

	err := row.Scan(
		&createdURL.OriginalURL,
		&createdURL.ShortURLCode,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		query := `
        SELECT original_url, short_url_code
        FROM shortened_urls
        WHERE original_url = $1
    	`

		row = r.pool.QueryRow(reqCtx, query, url.OriginalURL)
		err = row.Scan(
			&createdURL.OriginalURL,
			&createdURL.ShortURLCode,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.URL{}, domain.ErrShortURLCodeCollision
		}
	}
	if err != nil {
		return domain.URL{}, fmt.Errorf("failed to get original url: %w", err)
	}

	return createdURL, nil
}
