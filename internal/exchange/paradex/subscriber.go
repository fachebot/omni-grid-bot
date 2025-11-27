package paradex

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/fachebot/omni-grid-bot/internal/config"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"golang.org/x/sync/singleflight"
)

type ParadexSubscriber struct {
	ctx    context.Context
	cancel context.CancelFunc
	proxy  config.Sock5Proxy

	mutex     sync.Mutex
	wg        sync.WaitGroup
	sf        singleflight.Group
	userConns map[string]*ParadexWS
	stopped   atomic.Bool

	userOrdersInChan  chan exchange.UserOrders
	userOrdersOutChan chan exchange.UserOrders
}

func NewParadexSubscriber(proxy config.Sock5Proxy) *ParadexSubscriber {
	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &ParadexSubscriber{
		ctx:              ctx,
		cancel:           cancel,
		proxy:            proxy,
		userConns:        make(map[string]*ParadexWS),
		userOrdersInChan: make(chan exchange.UserOrders, 1024*8),
	}
	return subscriber
}

func (subscriber *ParadexSubscriber) Stop() {
	logger.Infof("[ParadexSubscriber] 准备停止服务")

	if !subscriber.stopped.CompareAndSwap(false, true) {
		logger.Warnf("[ParadexSubscriber] 服务已经停止")
		return
	}

	// 关闭所有连接
	subscriber.mutex.Lock()
	conns := make([]*ParadexWS, 0, len(subscriber.userConns))
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
	if subscriber.userOrdersOutChan != nil {
		close(subscriber.userOrdersOutChan)
		subscriber.userOrdersOutChan = nil
	}

	logger.Infof("[ParadexSubscriber] 服务已经停止")
}

func (subscriber *ParadexSubscriber) Start() {
	logger.Infof("[ParadexSubscriber] 开始运行服务")
	go subscriber.run()
}

func (subscriber *ParadexSubscriber) GetAccountOrdersChan() <-chan exchange.UserOrders {
	if subscriber.userOrdersOutChan == nil {
		subscriber.userOrdersOutChan = make(chan exchange.UserOrders, 1024*8)
	}
	return subscriber.userOrdersOutChan
}

func (subscriber *ParadexSubscriber) SubscribeAccountOrders(userClient *UserClient) error {
	account := userClient.DexAccount()

	_, err, _ := subscriber.sf.Do(account, func() (any, error) {
		subscriber.mutex.Lock()
		if _, exists := subscriber.userConns[account]; exists {
			subscriber.mutex.Unlock()
			return nil, nil
		}
		subscriber.mutex.Unlock()

		subscriber.wg.Add(1)
		ws := NewParadexWS(
			subscriber.ctx,
			userClient,
			subscriber.userOrdersInChan,
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

func (subscriber *ParadexSubscriber) UnsubscribeAccountOrders(userClient *UserClient) error {
	subscriber.mutex.Lock()

	ws, ok := subscriber.userConns[userClient.DexAccount()]
	if !ok {
		subscriber.mutex.Unlock()
		return nil
	}

	ws.Stop()
	delete(subscriber.userConns, userClient.DexAccount())

	subscriber.mutex.Unlock()

	return nil
}

func (subscriber *ParadexSubscriber) run() {
	for {
		select {
		case <-subscriber.ctx.Done():
			return
		case data := <-subscriber.userOrdersInChan:
			if subscriber.userOrdersOutChan != nil {
				subscriber.userOrdersOutChan <- data
			}
		}
	}
}

func (subscriber *ParadexSubscriber) onWsServiceStopped(dexAccount string) {
	subscriber.mutex.Lock()
	delete(subscriber.userConns, dexAccount)
	subscriber.mutex.Unlock()

	subscriber.wg.Done()
}
