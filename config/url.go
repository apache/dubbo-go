package config

import (
	"net/url"
	"time"
)

// Extension - URL
type URL interface {
	CheckMethod(string) bool
	PrimitiveURL() string
	Query() url.Values
	Location() string
	Timeout() time.Duration
	Group() string
	Protocol() string
	Version() string
	Ip() string
	Port() string
	Path() string
}
