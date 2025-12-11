package engine

import (
	"container/heap"
	"context"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/exchange/variational"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/samber/lo"
)

// OrderCancelled 订单取消回调函数类型
type OrderCancelled func(ctx context.Context, svcCtx *svc.ServiceContext, engine *StrategyEngine, s *ent.Strategy)

// Strategy 策略接口
type Strategy interface {
	Get() *ent.Strategy
	Update(s *ent.Strategy)
	OnOrdersChanged(ctx context.Context) error
}

// StrategyEngine 策略引擎
type StrategyEngine struct {
	// 上下文管理
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	// 依赖服务
	svcCtx           *svc.ServiceContext
	onOrderCancelled OrderCancelled

	// 交易所订阅器
	lighterSubscriber     *lighter.LighterSubscriber
	paradexSubscriber     *paradex.ParadexSubscriber
	variationalSubscriber *variational.VariationalSubscriber

	// 策略管理
	mutex           sync.RWMutex
	strategyMap     map[string]Strategy // 策略ID -> 策略实例
	userStrategyMap map[string][]string // 用户账户 -> 策略ID列表

	// 重试管理
	retryHeap *retryHeap            // 最小堆
	retrySet  map[string]*retryItem // 快速查找是否在重试队列
}

// NewStrategyEngine 创建策略引擎实例
func NewStrategyEngine(
	svcCtx *svc.ServiceContext,
	lighterSubscriber *lighter.LighterSubscriber,
	paradexSubscriber *paradex.ParadexSubscriber,
	variationalSubscriber *variational.VariationalSubscriber,
	onOrderCancelled OrderCancelled,
) *StrategyEngine {
	h := make(retryHeap, 0)
	heap.Init(&h)

	ctx, cancel := context.WithCancel(context.Background())
	return &StrategyEngine{
		ctx:                   ctx,
		cancel:                cancel,
		svcCtx:                svcCtx,
		onOrderCancelled:      onOrderCancelled,
		lighterSubscriber:     lighterSubscriber,
		paradexSubscriber:     paradexSubscriber,
		variationalSubscriber: variationalSubscriber,
		strategyMap:           make(map[string]Strategy),
		userStrategyMap:       make(map[string][]string),
		retryHeap:             &h,
		retrySet:              make(map[string]*retryItem),
	}
}

// Start 启动策略引擎
func (engine *StrategyEngine) Start() {
	if engine.stopChan != nil {
		return
	}

	engine.stopChan = make(chan struct{})
	logger.Infof("[StrategyEngine] 开始运行服务")
	go engine.run()
}

// Stop 停止策略引擎
func (engine *StrategyEngine) Stop() {
	if engine.stopChan == nil {
		return
	}

	logger.Infof("[StrategyEngine] 准备停止服务")

	engine.cancel()

	<-engine.stopChan
	close(engine.stopChan)
	engine.stopChan = nil

	logger.Infof("[StrategyEngine] 服务已经停止")
}

// StartStrategy 启动策略
func (engine *StrategyEngine) StartStrategy(s Strategy) error {
	record := s.Get()

	// 添加策略到引擎
	engine.addStrategyToEngine(record.GUID, record.Account, s)

	// 订阅用户订单
	return engine.subscribeUserOrders(record)
}

// StopStrategy 停止策略
func (engine *StrategyEngine) StopStrategy(id string) {
	// 从引擎中移除策略
	record, userStrategyCount := engine.removeStrategyFromEngine(id)
	if record == nil {
		return
	}

	// 如果用户没有其他策略，取消订阅
	if userStrategyCount == 0 {
		engine.unsubscribeUserOrders(record)
	}
}

// UpdateStrategy 更新策略
func (engine *StrategyEngine) UpdateStrategy(entStrategy *ent.Strategy) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	if s, ok := engine.strategyMap[entStrategy.GUID]; ok {
		s.Update(entStrategy)
	}
}

// addToRetryQueue 添加策略到重试队列
func (engine *StrategyEngine) addToRetryQueue(strategyID string, retryTime time.Time) {
	// 如果已经在队列中，先移除
	if item, exists := engine.retrySet[strategyID]; exists {
		heap.Remove(engine.retryHeap, item.index)
	}

	// 添加新的重试项
	item := &retryItem{
		strategyID: strategyID,
		retryTime:  retryTime,
	}
	heap.Push(engine.retryHeap, item)
	engine.retrySet[strategyID] = item

	logger.Debugf("[StrategyEngine] 策略已加入重试队列, id: %s, retryTime: %s",
		strategyID, retryTime.Format(time.RFC3339))
}

// removeFromRetryQueue 从重试队列移除策略
func (engine *StrategyEngine) removeFromRetryQueue(strategyID string) {
	if item, exists := engine.retrySet[strategyID]; exists {
		heap.Remove(engine.retryHeap, item.index)
		delete(engine.retrySet, strategyID)

		logger.Debugf("[StrategyEngine] 策略已从重试队列移除, id: %s", strategyID)
	}
}

// processRetries 处理重试队列
func (engine *StrategyEngine) processRetries() {
	for engine.retryHeap.Len() > 0 {
		item := (*engine.retryHeap)[0]
		if time.Now().Before(item.retryTime) {
			break
		}

		// 从堆中移除
		heap.Pop(engine.retryHeap)
		delete(engine.retrySet, item.strategyID)

		// 获取策略并重试
		engine.mutex.RLock()
		strategy, exists := engine.strategyMap[item.strategyID]
		engine.mutex.RUnlock()

		if !exists {
			logger.Warnf("[StrategyEngine] 重试时策略不存在, id: %s", item.strategyID)
			continue
		}

		logger.Infof("[StrategyEngine] 开始重试策略, id: %s", item.strategyID)
		engine.executeStrategy(strategy)
	}
}

// run 主运行循环
func (engine *StrategyEngine) run() {
	timer := time.NewTimer(0)
	defer timer.Stop()

	lighterChan := engine.lighterSubscriber.SubscriptionChan()
	paradexChan := engine.paradexSubscriber.SubscriptionChan()
	variationalChan := engine.variationalSubscriber.SubscriptionChan()

	for {
		var msg *exchange.SubMessage
		select {
		case <-timer.C:
			engine.processRetries()
			timer.Reset(time.Second * 1)

		case <-engine.ctx.Done():
			engine.stopChan <- struct{}{}
			return

		case data := <-lighterChan:
			msg = &data
		case data := <-paradexChan:
			msg = &data
		case data := <-variationalChan:
			msg = &data
		}

		if msg != nil {
			if msg.UserOrders != nil {
				engine.processOrders(*msg.UserOrders)
			}
		}
	}
}

// addStrategyToEngine 添加策略到引擎
func (engine *StrategyEngine) addStrategyToEngine(strategyID, account string, s Strategy) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	// 如果策略不存在，更新用户策略列表
	if _, exists := engine.strategyMap[strategyID]; !exists {
		userStrategyList := engine.userStrategyMap[account]
		if userStrategyList == nil {
			userStrategyList = make([]string, 0)
		}

		// 避免重复添加
		if _, found := lo.Find(userStrategyList, func(guid string) bool {
			return guid == strategyID
		}); !found {
			userStrategyList = append(userStrategyList, strategyID)
		}
		engine.userStrategyMap[account] = userStrategyList
	}

	engine.strategyMap[strategyID] = s
}

// removeStrategyFromEngine 从引擎中移除策略
func (engine *StrategyEngine) removeStrategyFromEngine(strategyID string) (*ent.Strategy, int) {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	s, ok := engine.strategyMap[strategyID]
	if !ok {
		return nil, 0
	}

	record := s.Get()
	userStrategyCount := 0

	// 更新用户策略列表
	if userStrategyList, ok := engine.userStrategyMap[record.Account]; ok {
		newList := make([]string, 0, len(userStrategyList))
		for _, guid := range userStrategyList {
			if guid != strategyID {
				newList = append(newList, guid)
			}
		}
		userStrategyCount = len(newList)
		engine.userStrategyMap[record.Account] = newList
	}

	delete(engine.strategyMap, strategyID)
	return record, userStrategyCount
}
