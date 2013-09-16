### LRUCache
A map + linked-list backed LRU cache. Cached items belong to a primary and secondary cache key. This allows you to generate multiple versions of the same object yet purge all variations. For example:

    // 100MB
    cache := lrucache.New(lrucache.Configure())
    cache.Set("video/43", ".json", video1)
    cache.Set("video/43", ".xml", video2)

    // will remove both entries from the cache
    cache.Remove("video/43")

The cache takes some liberties. For example, a delay prevents recently promoted items from being re-promoted (thus minimizing the occurrences of a global write lock on the linked list). Furthermore, the garbage collection keeps some artifacts around. We've found these compromises suitable for caching HTTP responses.

There are now a number of respectable caching libraries for Go, such as [GroupCache](https://github.com/golang/groupcache) and [Vitess](https://code.google.com/p/vitess/source/browse/go/cache/lru_cache.go).

The cache is thread-safe.

### Usage
Start by creating an instance of the cache:

    cache := lrucache.New(lrucache.Configure())

The cache `size`, `itemsToPrune` and `callback` can be specific via the `configuration` object:

    cache := lrucache.New(
          lrucache.Configure()
            .Size(1 * 1024 * 1024 * 1024)  //1GB
            .ItemsToPrune(5000)            //# of items to prune when a GC runs
            .Callback(func(){ atomic.AddUint64(&gcs, 1) }) //called whenever a GC runs
        )


After creating an instance, you can `Get`, `Set` or `Remove`. Items added to the cache must implement the `lrucache.CacheItem` interface, which simply defines `Debug() []byte` and `Expiry() time.Time`:


    type Response struct {
      body []byte
      statusCode int
    }
    func (r *Response) Debug() []byte {
      // Adds arbitrary data to the output of cache.Debug() for this line item.
      // In this case, returning len(body) might be useful, or just do nothing
      // if you aren't using Debug
      return []byte{}
    }
    func (r *Response) Expires() time.Time {
      return time.Now().Add(time.Hour)
    }

### Installation
Install using the "go get" command:

    go get github.com/viki-org/lrucache
