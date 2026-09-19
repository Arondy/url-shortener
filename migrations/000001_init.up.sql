CREATE TABLE shortened_urls (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    original_url TEXT NOT NULL UNIQUE,
    short_url_code TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
