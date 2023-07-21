# `objcache` package

[![codecov](https://codecov.io/gh/voedger/voedger/branch/main/graph/badge.svg?token=1O1pA6zdYs)](https://codecov.io/gh/voedger/voedger/objcache)

## Proposal

### Principles

1. Can not use [fastcache](ghttps://ithub.com/VictoriaMetrics/fastcache) or other caches that need serialization value to []byte.
2. We do not need caches with expiration time, like [bigcache](https://github.com/allegro/bigcache)
3. Prefer LRU or MRU (or balanced both way strategy) cache with limitation by items count.
4. It would be nice if the cache allowed you to track the eviction of elements from it. This can be useful for returning resources occupied by the evicted event to the pool.
5. Do not care about metrics so far (will be added later)
6. Event.Release() should track all clients (projectors/cache etc., kind of AddRef(), AddRef(), Release() from projectors, Release from cache)

### Candidates

1. [Ristretto, 4.8K stars](https://github.com/dgraph-io/ristretto)
1. [hashicorp/golang-lru, 3.6K starts](https://github.com/hashicorp/golang-lru)
1. [gcache, 2.4K stars](https://github.com/bluele/gcache)
1. [theine-go, 103 stars](https://github.com/Yiling-J/theine-go)
1. [imcache, 66 stars](https://github.com/erni27/imcache)
1. [floatdrop, 19 stars](https://github.com/floatdrop/lru)

- *theine-go* seems as a technical leader, but not popular (yet?)
- *theine-go* [claims](https://github.com/dgraph-io/ristretto/issues/336) that *Ristretto* hits ration is very low (???). Strange that the *Ristretto* Team has not answered it yet
- *Ristretto* and *gcache* does NOT provide generic interface
- *theine-go*, *hashicorp/golang-lru*, *imcache* and *floatdrop* DOES provide generic interface
- *hashicorp/golang-lru* seems as a preferable solution (generic + popular)
  - LRU, since it is not clear how to handle event eviction in MRU version

## Technical Design

- new package `objcache` which encapsulate calls to lru cache
  - will be easy to replace the implementation
- `objcache` is a separate package since it is need to cache tokens

`objcache` interface

```golang
func New[Key comparable, Value any](size int, onEvict func (K, V)) ICache[K, V]

type ICache[Key comparable, Value any] interface {
  Put(K, V)
  Get(K) (value V, ok bool)
}
```
