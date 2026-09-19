package domain

import "errors"

var ErrShortURLCodeNotFound = errors.New("provided short url code not found")
var ErrShortURLCodeCollision = errors.New("short url code collision")
