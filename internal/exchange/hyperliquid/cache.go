package hyperliquid

import (
	"context"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

type Cache struct {
	client       *Client
	mutex        sync.Mutex
	meta         *Meta
	assetCtxs    map[int]*AssetContext
	mids         map[string]string
	accountCache *cache.Cache
	orderCache   *cache.Cache
}

func NewCache(client *Client) *Cache {
	return &Cache{
		client:       client,
		assetCtxs:    make(map[int]*AssetContext),
		mids:         make(map[string]string),
		accountCache: cache.New(10*time.Second, 30*time.Second),
		orderCache:   cache.New(5*time.Second, 15*time.Second),
	}
}

func (c *Cache) EnsureLoadMeta(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.meta != nil {
		return nil
	}

	meta, err := c.client.Meta(ctx)
	if err != nil {
		return err
	}

	c.meta = meta
	return nil
}

func (c *Cache) EnsureLoadAssetCtxs(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.assetCtxs) > 0 {
		return nil
	}

	metaAndCtxs, err := c.client.MetaAndAssetCtxs(ctx)
	if err != nil {
		return err
	}

	c.meta = &metaAndCtxs.Meta
	for i, ctx := range metaAndCtxs.AssetCtxs {
		c.assetCtxs[i+1] = &ctx
	}

	return nil
}

func (c *Cache) EnsureLoadMids(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if len(c.mids) > 0 {
		return nil
	}

	mids, err := c.client.AllMids(ctx)
	if err != nil {
		return err
	}

	for k, v := range mids {
		c.mids[k] = v.String()
	}

	return nil
}

func (c *Cache) GetMeta(ctx context.Context) (*Meta, error) {
	if err := c.EnsureLoadMeta(ctx); err != nil {
		return nil, err
	}
	return c.meta, nil
}

func (c *Cache) GetAssetCtx(ctx context.Context, asset int) (*AssetContext, error) {
	if err := c.EnsureLoadAssetCtxs(ctx); err != nil {
		return nil, err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	assetCtx, ok := c.assetCtxs[asset]
	if !ok {
		return nil, nil
	}
	return assetCtx, nil
}

func (c *Cache) GetMid(ctx context.Context, coin string) (string, error) {
	if err := c.EnsureLoadMids(ctx); err != nil {
		return "", err
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	mid, ok := c.mids[coin]
	if !ok {
		return "", nil
	}
	return mid, nil
}

func (c *Cache) GetCoinByAsset(asset int) string {
	if c.meta == nil {
		return ""
	}
	if asset <= 0 || asset > len(c.meta.Universe) {
		return ""
	}
	return c.meta.Universe[asset-1].Name
}

func (c *Cache) GetAssetByCoin(coin string) int {
	if c.meta == nil {
		return 0
	}
	for i, u := range c.meta.Universe {
		if u.Name == coin {
			return i + 1
		}
	}
	return 0
}

func (c *Cache) GetUserState(ctx context.Context, address string) (*AccountState, error) {
	cached, found := c.accountCache.Get(address)
	if found {
		return cached.(*AccountState), nil
	}

	state, err := c.client.UserState(ctx, address)
	if err != nil {
		return nil, err
	}

	c.accountCache.Set(address, state, cache.DefaultExpiration)
	return state, nil
}

func (c *Cache) GetOpenOrders(ctx context.Context, address string) ([]Order, error) {
	cached, found := c.orderCache.Get(address)
	if found {
		return cached.([]Order), nil
	}

	orders, err := c.client.OpenOrders(ctx, address)
	if err != nil {
		return nil, err
	}

	c.orderCache.Set(address, orders, cache.DefaultExpiration)
	return orders, nil
}

func (c *Cache) InvalidateAccountCache(address string) {
	c.accountCache.Delete(address)
}

func (c *Cache) InvalidateOrderCache(address string) {
	c.orderCache.Delete(address)
}
