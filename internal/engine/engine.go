package engine

import (
	"container/heap"
	"context"
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/exchange/variational"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	gridstrategy "github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
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

	lighterOrdersChan := engine.lighterSubscriber.GetAccountOrdersChan()
	paradexOrdersChan := engine.paradexSubscriber.GetAccountOrdersChan()
	variationalOrdersChan := engine.variationalSubscriber.GetAccountOrdersChan()

	for {
		select {
		case <-timer.C:
			engine.processRetries()
			timer.Reset(time.Second * 1)

		case <-engine.ctx.Done():
			engine.stopChan <- struct{}{}
			return

		case data := <-lighterOrdersChan:
			engine.processLighterOrders(data)

		case data := <-paradexOrdersChan:
			engine.processOrders(data)

		case data := <-variationalOrdersChan:
			engine.processOrders(data)
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

// subscribeUserOrders 订阅用户订单
func (engine *StrategyEngine) subscribeUserOrders(record *ent.Strategy) error {
	switch record.Exchange {
	case exchange.Lighter:
		return engine.subscribeLighterOrders(record)
	case exchange.Paradex:
		return engine.subscribeParadexOrders(record)
	case exchange.Variational:
		return engine.subscribeVariationalOrders(record)
	default:
		return errors.New("exchange unsupported")
	}
}

// unsubscribeUserOrders 取消订阅用户订单
func (engine *StrategyEngine) unsubscribeUserOrders(record *ent.Strategy) {
	switch record.Exchange {
	case exchange.Lighter:
		engine.unsubscribeLighterOrders(record)
	case exchange.Paradex:
		engine.unsubscribeParadexOrders(record)
	case exchange.Variational:
		engine.unsubscribeVariationalOrders(record)
	}
}

// subscribeLighterOrders 订阅 Lighter 订单
func (engine *StrategyEngine) subscribeLighterOrders(record *ent.Strategy) error {
	signer, err := helper.GetLighterClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
		return err
	}

	if err = engine.lighterSubscriber.SubscribeAccountOrders(signer); err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
	}
	return err
}

// unsubscribeLighterOrders 取消订阅 Lighter 订单
func (engine *StrategyEngine) unsubscribeLighterOrders(record *ent.Strategy) {
	signer, err := helper.GetLighterClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, %v", err)
		return
	}

	if err = engine.lighterSubscriber.UnsubscribeAccountOrders(signer); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
	}
}

// subscribeParadexOrders 订阅 Paradex 订单
func (engine *StrategyEngine) subscribeParadexOrders(record *ent.Strategy) error {
	userClient, err := helper.GetParadexClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %s, %v", userClient.DexAccount(), err)
		return err
	}

	if err = engine.paradexSubscriber.SubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %s, %v", userClient.DexAccount(), err)
	}
	return err
}

// unsubscribeParadexOrders 取消订阅 Paradex 订单
func (engine *StrategyEngine) unsubscribeParadexOrders(record *ent.Strategy) {
	userClient, err := helper.GetParadexClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, %v", err)
		return
	}

	account := userClient.DexAccount()
	if err = engine.paradexSubscriber.UnsubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %s, %v", account, err)
	}
}

// subscribeVariationalOrders 订阅 Variational 订单
func (engine *StrategyEngine) subscribeVariationalOrders(record *ent.Strategy) error {
	userClient, err := helper.GetVariationalClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %s, %v", userClient.EthAccount(), err)
		return err
	}

	if err = engine.variationalSubscriber.SubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %s, %v", userClient.EthAccount(), err)
	}
	return err
}

// unsubscribeVariationalOrders 取消订阅 Variational 订单
func (engine *StrategyEngine) unsubscribeVariationalOrders(record *ent.Strategy) {
	userClient, err := helper.GetVariationalClient(engine.svcCtx, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, %v", err)
		return
	}

	account := userClient.EthAccount()
	if err = engine.variationalSubscriber.UnsubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %s, %v", account, err)
	}
}

// processLighterOrders 处理 Lighter 订单
func (engine *StrategyEngine) processLighterOrders(userOrders exchange.UserOrders) {
	orders, err := engine.parseLighterOrders(userOrders)
	if err != nil {
		return
	}

	if err = engine.handleUserOrders(userOrders, orders); err != nil {
		engine.resubscribe(userOrders.Account)
	}
}

// processOrders 处理订单
func (engine *StrategyEngine) processOrders(userOrders exchange.UserOrders) {
	orders, err := engine.toEntOrders(userOrders)
	if err != nil {
		return
	}

	if err = engine.handleUserOrders(userOrders, orders); err != nil {
		engine.resubscribe(userOrders.Account)
	}
}

// resubscribe 重新订阅
func (engine *StrategyEngine) resubscribe(account string) {
	userStrategyList := engine.getUserStrategyList(account)
	if len(userStrategyList) == 0 {
		return
	}

	// 只需要重新订阅一次（所有策略共享同一个账户订阅）
	firstStrategy := userStrategyList[0].Get()

	switch firstStrategy.Exchange {
	case exchange.Lighter:
		engine.resubscribeLighter(firstStrategy)
	case exchange.Paradex:
		engine.resubscribeParadex(firstStrategy)
	case exchange.Variational:
		engine.resubscribeVariational(firstStrategy)
	}
}

// resubscribeLighter 重新订阅 Lighter
func (engine *StrategyEngine) resubscribeLighter(strategy *ent.Strategy) {
	signer, err := helper.GetLighterClient(engine.svcCtx, strategy)
	if err != nil {
		logger.Warnf("[StrategyEngine] 获取客户端失败, %v", err)
		return
	}

	accountIndex := signer.GetAccountIndex()

	// 取消订阅
	if err = engine.lighterSubscriber.UnsubscribeAccountOrders(signer); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", accountIndex, err)
	}

	// 重新订阅
	if err = engine.lighterSubscriber.SubscribeAccountOrders(signer); err != nil {
		logger.Warnf("[StrategyEngine] 重新订阅账户订单活动失败, account: %d, %v", accountIndex, err)
	}
}

// resubscribeParadex 重新订阅 Paradex
func (engine *StrategyEngine) resubscribeParadex(strategy *ent.Strategy) {
	userClient, err := helper.GetParadexClient(engine.svcCtx, strategy)
	if err != nil {
		logger.Warnf("[StrategyEngine] 获取客户端失败, account: %s, %v", strategy.Account, err)
		return
	}

	account := userClient.DexAccount()

	// 取消订阅
	if err = engine.paradexSubscriber.UnsubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %s, %v", account, err)
	}

	// 重新订阅
	if err = engine.paradexSubscriber.SubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 重新订阅账户订单活动失败, account: %s, %v", account, err)
	}
}

// resubscribeVariational 重新订阅 Variational
func (engine *StrategyEngine) resubscribeVariational(strategy *ent.Strategy) {
	userClient, err := helper.GetVariationalClient(engine.svcCtx, strategy)
	if err != nil {
		logger.Warnf("[StrategyEngine] 获取客户端失败, account: %s, %v", strategy.Account, err)
		return
	}

	account := userClient.EthAccount()

	// 取消订阅
	if err = engine.variationalSubscriber.UnsubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %s, %v", account, err)
	}

	// 重新订阅
	if err = engine.variationalSubscriber.SubscribeAccountOrders(userClient); err != nil {
		logger.Warnf("[StrategyEngine] 重新订阅账户订单活动失败, account: %s, %v", account, err)
	}
}

// getUserStrategyList 获取用户策略列表
func (engine *StrategyEngine) getUserStrategyList(account string) []Strategy {
	engine.mutex.Lock()
	defer engine.mutex.Unlock()

	userStrategyIds, ok := engine.userStrategyMap[account]
	if !ok {
		return nil
	}

	userStrategyList := make([]Strategy, 0, len(userStrategyIds))
	for _, guid := range userStrategyIds {
		if s, ok := engine.strategyMap[guid]; ok {
			userStrategyList = append(userStrategyList, s)
		}
	}

	return userStrategyList
}

// syncUserOrders 同步用户订单
func (engine *StrategyEngine) syncUserOrders(userOrders exchange.UserOrders, newOrders []ent.Order) error {
	logger.Debugf("[StrategyEngine] 同步用户订单开始, account: %s", userOrders.Account)

	// 查询账户策略
	userStrategyList := engine.getUserStrategyList(userOrders.Account)
	if len(userStrategyList) == 0 {
		logger.Debugf("[StrategyEngine] 账户没有正在运行的策略, account: %s", userOrders.Account)
		return nil
	}

	// 如果是快照数据，需要同步订单
	if userOrders.IsSnapshot {
		if err := engine.synchronizeOrders(userOrders.Account, userStrategyList); err != nil {
			return err
		}
	}

	// 更新新订单数据
	if err := engine.saveOrders(newOrders); err != nil {
		logger.Errorf("[StrategyEngine] 保存用户活跃订单失败, %v", err)
		return err
	}

	logger.Debugf("[StrategyEngine] 同步用户订单结束, account: %s", userOrders.Account)
	return nil
}

// synchronizeOrders 同步订单数据
func (engine *StrategyEngine) synchronizeOrders(account string, strategies []Strategy) error {
	for _, s := range strategies {
		adapter, err := helper.NewExchangeAdapterFromStrategy(engine.svcCtx, s.Get())
		if err != nil {
			logger.Errorf("[StrategyEngine] 创建交易所适配器失败, account: %s, exchange: %s, %v",
				s.Get().Account, s.Get().Exchange, err)
			continue
		}

		if err = adapter.SyncUserOrders(engine.ctx); err != nil {
			logger.Errorf("[StrategyEngine] 同步用户订单失败, account: %s, %v", account, err)
			continue
		}

		return nil // 同步成功
	}

	logger.Errorf("[StrategyEngine] 无法完成同步用户订单, account: %s", account)
	return errors.New("synchronization failed")
}

// saveOrders 保存订单
func (engine *StrategyEngine) saveOrders(orders []ent.Order) error {
	return util.Tx(engine.ctx, engine.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, order := range orders {
			if err := engine.svcCtx.OrderModel.Upsert(engine.ctx, order); err != nil {
				return err
			}
		}
		return nil
	})
}

// toEntOrders 转换为 Ent 订单
func (engine *StrategyEngine) toEntOrders(userOrders exchange.UserOrders) ([]ent.Order, error) {
	entOrders := make([]ent.Order, 0, len(userOrders.Orders))

	for _, ord := range userOrders.Orders {
		entOrders = append(entOrders, ent.Order{
			Exchange:          userOrders.Exchange,
			Account:           userOrders.Account,
			Symbol:            ord.Symbol,
			Side:              ord.Side,
			Price:             ord.Price,
			OrderId:           ord.OrderID,
			ClientOrderId:     ord.ClientOrderID,
			BaseAmount:        ord.BaseAmount,
			FilledBaseAmount:  ord.FilledBaseAmount,
			FilledQuoteAmount: ord.FilledQuoteAmount,
			Status:            ord.Status,
			Timestamp:         ord.Timestamp,
		})
	}

	return entOrders, nil
}

// parseLighterOrders 解析 Lighter 订单
func (engine *StrategyEngine) parseLighterOrders(userOrders exchange.UserOrders) ([]ent.Order, error) {
	// 转换市场ID为交易对符号
	for i := range userOrders.Orders {
		marketIndex, err := strconv.ParseInt(userOrders.Orders[i].Symbol, 10, 64)
		if err != nil {
			logger.Fatalf("[StrategyEngine] 解析市场ID失败, account: %s, symbol: %s",
				userOrders.Account, userOrders.Orders[i].Symbol)
			return nil, err
		}

		symbol, err := engine.svcCtx.LighterCache.GetSymbolByMarketId(engine.ctx, uint8(marketIndex))
		if err != nil {
			logger.Fatalf("[StrategyEngine] 查询市场代币符号失败, account: %s, marketIndex: %d",
				userOrders.Account, marketIndex)
			return nil, err
		}

		userOrders.Orders[i].Symbol = symbol
	}

	return engine.toEntOrders(userOrders)
}

// handleUserOrders 处理用户订单
func (engine *StrategyEngine) handleUserOrders(userOrders exchange.UserOrders, newOrders []ent.Order) error {
	// 同步用户订单
	if err := engine.syncUserOrders(userOrders, newOrders); err != nil {
		logger.Errorf("[StrategyEngine] 同步用户订单失败, account: %s, %v", userOrders.Account, err)
		return err
	}

	// 执行用户策略
	userStrategyList := engine.getUserStrategyList(userOrders.Account)
	for _, strategy := range userStrategyList {
		engine.executeStrategy(strategy)
	}

	return nil
}

// executeStrategy 执行策略
func (engine *StrategyEngine) executeStrategy(strategy Strategy) {
	s := strategy.Get()
	logger.Debugf("[StrategyEngine] 执行用户策略开始, id: %s, account: %s, symbol: %s",
		s.GUID, s.Account, s.Symbol)

	err := strategy.OnOrdersChanged(engine.ctx)
	if err != nil {
		engine.addToRetryQueue(s.GUID, time.Now().Add(15*time.Second))

		// 处理订单取消错误
		if errors.Is(err, gridstrategy.ErrOrderCanceled) && engine.onOrderCancelled != nil {
			engine.onOrderCancelled(engine.ctx, engine.svcCtx, engine, s)
		}

		logger.Errorf("[StrategyEngine] 执行用户策略失败, id: %s, account: %s, symbol: %s, %v",
			s.GUID, s.Account, s.Symbol, err)
		return
	}

	engine.removeFromRetryQueue(s.GUID)

	logger.Debugf("[StrategyEngine] 执行用户策略结束, id: %s, account: %s, symbol: %s",
		s.GUID, s.Account, s.Symbol)
}
