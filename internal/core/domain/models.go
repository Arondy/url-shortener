package domain

import "time"

type URL struct {
	OriginalURL  string
	ShortURLCode string
	CreatedAt    time.Time
}
