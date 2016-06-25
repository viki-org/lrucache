package lrucache

import (
	"time"
)

type CacheItem interface {
	Expires() time.Time
	Debug() []byte
}
