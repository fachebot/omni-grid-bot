package cache

import (
	"context"
	"errors"
	"sync"

	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
)

type LighterCache struct {
	client     *lighter.Client
	mutex      sync.Mutex
	markets    map[uint8]string
	orderBooks map[string]*lighter.OrderBookMetadata
}

func NewLighterCache(client *lighter.Client) *LighterCache {
	return &LighterCache{
		client:     client,
		markets:    map[uint8]string{},
		orderBooks: make(map[string]*lighter.OrderBookMetadata),
	}
}

func (cache *LighterCache) GetSymbolByMarketId(ctx context.Context, marketIndex uint8) (string, error) {
	err := cache.ensureLoadCache(ctx)
	if err != nil {
		return "", err
	}

	cache.mutex.Lock()
	symbol, ok := cache.markets[marketIndex]
	if ok {
		cache.mutex.Unlock()
		return symbol, nil
	}
	cache.mutex.Unlock()

	return "", errors.New("not found")
}

func (cache *LighterCache) GetOrderBookMetadata(ctx context.Context, symbol string) (*lighter.OrderBookMetadata, error) {
	err := cache.ensureLoadCache(ctx)
	if err != nil {
		return nil, err
	}

	cache.mutex.Lock()
	metadata, ok := cache.orderBooks[symbol]
	if ok {
		cache.mutex.Unlock()
		return metadata, nil
	}
	cache.mutex.Unlock()

	return nil, errors.New("not found")
}

func (cache *LighterCache) ensureLoadCache(ctx context.Context) error {
	cache.mutex.Lock()
	if len(cache.orderBooks) > 0 {
		cache.mutex.Unlock()
		return nil
	}
	cache.mutex.Unlock()

	result, err := cache.client.GetOrderBooksMetadata(ctx)
	if err != nil {
		return err
	}

	cache.mutex.Lock()
	for _, medadata := range result.OrderBooks {
		cache.orderBooks[medadata.Symbol] = medadata
		cache.markets[medadata.MarketID] = medadata.Symbol
	}
	cache.mutex.Unlock()

	return nil
}
