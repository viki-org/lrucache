package lrucache

import (
  "time"
)

type CacheItem interface {
  Size() int64
  Expires() time.Time
}
