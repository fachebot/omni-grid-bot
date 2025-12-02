package cache

import (
	"fmt"
	"sync"
)

type PendingOrdersCache struct {
	rb            sync.RWMutex
	pendingOrders map[string]map[string]struct{}
}

func NewPendingOrdersCache() *PendingOrdersCache {
	return &PendingOrdersCache{pendingOrders: make(map[string]map[string]struct{})}
}

func (c *PendingOrdersCache) Add(exchange, user, clientOrderId string) {
	key := fmt.Sprintf("%s:%s", exchange, user)

	c.rb.Lock()
	defer c.rb.Unlock()

	set, ok := c.pendingOrders[key]
	if !ok {
		set = make(map[string]struct{})
	}
	set[clientOrderId] = struct{}{}
	c.pendingOrders[key] = set
}

func (c *PendingOrdersCache) Del(exchange, user, clientOrderId string) {
	key := fmt.Sprintf("%s:%s", exchange, user)

	c.rb.Lock()
	defer c.rb.Unlock()

	set, ok := c.pendingOrders[key]
	if !ok {
		return
	}
	delete(set, clientOrderId)
	c.pendingOrders[key] = set
}

func (c *PendingOrdersCache) List(exchange, user string) []string {
	key := fmt.Sprintf("%s:%s", exchange, user)

	c.rb.RLock()
	defer c.rb.RUnlock()

	set, ok := c.pendingOrders[key]
	if !ok {
		return nil
	}

	orders := make([]string, 0, len(set))
	for orderId := range set {
		orders = append(orders, orderId)
	}
	return orders
}

func (c *PendingOrdersCache) Exist(exchange, user, clientOrderId string) bool {
	key := fmt.Sprintf("%s:%s", exchange, user)

	c.rb.RLock()
	defer c.rb.RUnlock()

	set, ok := c.pendingOrders[key]
	if !ok {
		return false
	}
	_, exist := set[clientOrderId]
	return exist
}
