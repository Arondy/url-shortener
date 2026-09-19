package domain

import "errors"

var ErrShortURLCodeNotFound = errors.New("provided short url code not found")
var ErrShortURLCodeExpired = errors.New("provided short url code expired")
var ErrShortURLCodeCollision = errors.New("short url code collision")
