package variational

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/fachebot/omni-grid-bot/internal/cache"
	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"golang.org/x/sync/singleflight"
)

type VariationalSubscriber struct {
	ctx    context.Context
	cancel context.CancelFunc
	proxy  config.Sock5Proxy

	mutex     sync.Mutex
	wg        sync.WaitGroup
	sf        singleflight.Group
	userConns map[string]*VariationalWS
	stopped   atomic.Bool

	subMsgChan         chan exchange.SubMessage
	userOrdersInChan   chan exchange.UserOrders
	pendingOrdersCache *cache.PendingOrdersCache
}

func NewVariationalSubscriber(pendingOrdersCache *cache.PendingOrdersCache, proxy config.Sock5Proxy) *VariationalSubscriber {
	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &VariationalSubscriber{
		ctx:                ctx,
		cancel:             cancel,
		proxy:              proxy,
		userConns:          make(map[string]*VariationalWS),
		userOrdersInChan:   make(chan exchange.UserOrders, 1024*8),
		pendingOrdersCache: pendingOrdersCache,
	}
	return subscriber
}

func (subscriber *VariationalSubscriber) Stop() {
	logger.Infof("[VariationalSubscriber] 准备停止服务")

	if !subscriber.stopped.CompareAndSwap(false, true) {
		logger.Warnf("[VariationalSubscriber] 服务已经停止")
		return
	}

	// 关闭所有连接
	subscriber.mutex.Lock()
	conns := make([]*VariationalWS, 0, len(subscriber.userConns))
	for _, conn := range subscriber.userConns {
		conns = append(conns, conn)
	}
	subscriber.mutex.Unlock()

	for _, conn := range conns {
		conn.Stop()
	}
	subscriber.wg.Wait()

	// 清理服务资源
	subscriber.cancel()
	close(subscriber.userOrdersInChan)
	if subscriber.subMsgChan != nil {
		close(subscriber.subMsgChan)
		subscriber.subMsgChan = nil
	}

	logger.Infof("[VariationalSubscriber] 服务已经停止")
}

func (subscriber *VariationalSubscriber) Start() {
	logger.Infof("[VariationalSubscriber] 开始运行服务")
	go subscriber.run()
}

func (subscriber *VariationalSubscriber) SubscriptionChan() <-chan exchange.SubMessage {
	if subscriber.subMsgChan == nil {
		subscriber.subMsgChan = make(chan exchange.SubMessage, 1024*8)
	}
	return subscriber.subMsgChan
}

func (subscriber *VariationalSubscriber) SubscribeAccountOrders(userClient *UserClient) error {
	account := userClient.EthAccount()

	_, err, _ := subscriber.sf.Do(account, func() (any, error) {
		subscriber.mutex.Lock()
		if _, ok := subscriber.userConns[account]; ok {
			subscriber.mutex.Unlock()
			return nil, nil
		}
		subscriber.mutex.Unlock()

		subscriber.wg.Add(1)
		ws := NewVariationalWS(
			subscriber.ctx,
			userClient,
			subscriber.userOrdersInChan,
			subscriber.pendingOrdersCache,
			subscriber.proxy,
			subscriber.onWsServiceStopped,
		)
		ws.Start()

		subscriber.mutex.Lock()
		subscriber.userConns[account] = ws
		subscriber.mutex.Unlock()

		return nil, nil
	})

	return err
}

func (subscriber *VariationalSubscriber) UnsubscribeAccountOrders(userClient *UserClient) error {
	account := userClient.EthAccount()

	_, err, _ := subscriber.sf.Do(account, func() (any, error) {
		subscriber.mutex.Lock()
		ws, ok := subscriber.userConns[userClient.EthAccount()]
		if !ok {
			subscriber.mutex.Unlock()
			return nil, nil
		}
		delete(subscriber.userConns, userClient.EthAccount())
		subscriber.mutex.Unlock()

		ws.Stop()

		return nil, nil
	})

	return err
}

func (subscriber *VariationalSubscriber) run() {
	for {
		select {
		case <-subscriber.ctx.Done():
			return
		case data := <-subscriber.userOrdersInChan:
			if subscriber.subMsgChan != nil {
				subscriber.subMsgChan <- exchange.SubMessage{UserOrders: &data}
			}
		}
	}
}

func (subscriber *VariationalSubscriber) onWsServiceStopped(dexAccount string) {
	subscriber.mutex.Lock()
	delete(subscriber.userConns, dexAccount)
	subscriber.mutex.Unlock()

	subscriber.wg.Done()
}
