package cache

import (
	"fmt"
	"time"

	gocache "github.com/patrickmn/go-cache"
)

type RecentOrdersCache struct {
	cache *gocache.Cache
}

func NewRecentOrdersCache() *RecentOrdersCache {
	return &RecentOrdersCache{cache: gocache.New(5*time.Minute, 10*time.Minute)}
}

func (router *RecentOrdersCache) Add(exchange, user, clientOrderId string) {
	key := fmt.Sprintf("%s:%s:%s", exchange, user, clientOrderId)
	router.cache.Add(key, "", gocache.DefaultExpiration)
}

func (router *RecentOrdersCache) Del(exchange, user, clientOrderId string) {
	key := fmt.Sprintf("%s:%s:%s", exchange, user, clientOrderId)
	router.cache.Delete(key)
}

func (router *RecentOrdersCache) Has(exchange, user, clientOrderId string) bool {
	key := fmt.Sprintf("%s:%s:%s", exchange, user, clientOrderId)
	_, ok := router.cache.Get(key)
	return ok
}
