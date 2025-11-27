package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
)

type ParadexCache struct {
	client  *paradex.Client
	mutex   sync.Mutex
	markets map[string]*paradex.Market
}

func NewParadexCache(client *paradex.Client) *ParadexCache {
	return &ParadexCache{
		client:  client,
		markets: make(map[string]*paradex.Market),
	}
}

func (cache *ParadexCache) GetMarketMetadata(ctx context.Context, market string) (*paradex.Market, error) {
	err := cache.ensureLoadCache(ctx)
	if err != nil {
		return nil, err
	}

	cache.mutex.Lock()
	metadata, ok := cache.markets[market]
	if ok {
		cache.mutex.Unlock()
		return metadata, nil
	}
	cache.mutex.Unlock()

	return nil, errors.New("not found")
}

func (cache *ParadexCache) ensureLoadCache(ctx context.Context) error {
	cache.mutex.Lock()
	if len(cache.markets) > 0 {
		cache.mutex.Unlock()
		return nil
	}
	cache.mutex.Unlock()

	result, err := cache.client.GetMarkets(ctx)
	if err != nil {
		return err
	}

	cache.mutex.Lock()
	for _, medadata := range result.Results {
		cache.markets[medadata.Symbol] = medadata
	}
	cache.mutex.Unlock()

	return nil
}
