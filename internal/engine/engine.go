package engine

import (
	"context"
	"errors"
	"strconv"
	"sync"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/perp-dex-grid-bot/internal/helper"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/samber/lo"
)

type Strategy interface {
	Get() *ent.Strategy
	OnUpdate(ctx context.Context) error
}

type StrategyEngine struct {
	ctx      context.Context
	cancel   context.CancelFunc
	stopChan chan struct{}

	svcCtx            *svc.ServiceContext
	lighterSubscriber *lighter.LighterSubscriber

	mutex           sync.RWMutex
	strategyMap     map[string]Strategy
	userStrategyMap map[string][]string
}

func NewStrategyEngine(svcCtx *svc.ServiceContext, lighterSubscriber *lighter.LighterSubscriber) *StrategyEngine {
	ctx, cancel := context.WithCancel(context.Background())
	return &StrategyEngine{
		ctx:               ctx,
		cancel:            cancel,
		svcCtx:            svcCtx,
		lighterSubscriber: lighterSubscriber,
		strategyMap:       make(map[string]Strategy),
		userStrategyMap:   make(map[string][]string),
	}
}

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

func (engine *StrategyEngine) Start() {
	if engine.stopChan != nil {
		return
	}

	engine.stopChan = make(chan struct{})

	logger.Infof("[StrategyEngine] 开始运行服务")

	go engine.run()
}

func (engine *StrategyEngine) StopStrategy(id string) {
	// 移除策略
	engine.mutex.Lock()

	s, ok := engine.strategyMap[id]
	if !ok {
		engine.mutex.Unlock()
		return
	}

	// 更新用户策略列表
	record := s.Get()
	userStrategyCount := 0
	userStrategyList, ok := engine.userStrategyMap[record.Account]
	if ok {
		newUserStrategyList := make([]string, 0, len(userStrategyList))
		for _, guid := range userStrategyList {
			if id != guid {
				newUserStrategyList = append(newUserStrategyList, guid)
			}
		}

		userStrategyCount = len(newUserStrategyList)
		engine.userStrategyMap[record.Account] = newUserStrategyList
	}
	engine.mutex.Unlock()

	// 取消订阅用户订单
	if userStrategyCount == 0 {
		switch record.Exchange {
		case exchange.Lighter:
			signer, err := helper.GetLighterClient(engine.svcCtx, record)
			if err != nil {
				logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			}

			err = engine.lighterSubscriber.UnsubscribeAccountOrders(signer)
			if err != nil {
				logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			}
		}
	}
}

func (engine *StrategyEngine) StartStrategy(s Strategy) (err error) {
	record := s.Get()

	// 添加策略
	engine.mutex.Lock()

	_, ok := engine.strategyMap[record.GUID]
	if !ok {
		// 更新用户策略列表
		userStrategyList, ok := engine.userStrategyMap[record.Account]
		if !ok {
			userStrategyList = make([]string, 0)
		}

		_, find := lo.Find(userStrategyList, func(guid string) bool {
			return guid == record.GUID
		})
		if !find {
			userStrategyList = append(userStrategyList, record.GUID)
		}
		engine.userStrategyMap[record.Account] = userStrategyList
	}
	engine.strategyMap[record.GUID] = s

	engine.mutex.Unlock()

	// 订阅用户订单
	switch record.Exchange {
	case exchange.Lighter:
		signer, err := helper.GetLighterClient(engine.svcCtx, record)
		if err != nil {
			logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			return err
		}

		err = engine.lighterSubscriber.SubscribeAccountOrders(signer)
		if err != nil {
			logger.Warnf("[StrategyEngine] 订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
		}
		return err
	default:
		return errors.New("exchange unsupported")
	}
}

func (engine *StrategyEngine) run() {
	lightUserOrdersChan := engine.lighterSubscriber.GetAccountOrdersChan()

	for {
		select {
		case <-engine.ctx.Done():
			engine.stopChan <- struct{}{}
			return
		case data := <-lightUserOrdersChan:
			orders, err := engine.parseLightOrders(data.Account, data.Orders)
			if err != nil {
				continue
			}

			// 处理用户订单，错误时重新同步订单数据
			err = engine.handleUserOrders(data, orders)
			if err != nil {
				engine.resubscribe(data.Account)
			}
		}
	}
}

func (engine *StrategyEngine) resubscribe(account string) {
	userStrategyList := engine.userStrategyList(account)
	if len(userStrategyList) == 0 {
		return
	}

	for _, s := range userStrategyList {
		switch s.Get().Exchange {
		case exchange.Lighter:
			signer, err := helper.GetLighterClient(engine.svcCtx, s.Get())
			if err != nil {
				logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			}

			err = engine.lighterSubscriber.UnsubscribeAccountOrders(signer)
			if err != nil {
				logger.Warnf("[StrategyEngine] 取消订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			}

			err = engine.lighterSubscriber.SubscribeAccountOrders(signer)
			if err != nil {
				logger.Warnf("[StrategyEngine] 重新订阅账户订单活动失败, account: %d, %v", signer.GetAccountIndex(), err)
			}

			return
		}
	}
}

func (engine *StrategyEngine) syncUserOrders(userOrders exchange.UserOrders, newOrders []ent.Order) error {
	// 查询账户策略
	userStrategyList := engine.userStrategyList(userOrders.Account)
	if len(userStrategyList) == 0 {
		logger.Debugf("[StrategyEngine] 账户没有正在运行的策略, account: %s", userOrders.Account)
		return nil
	}

	// 同步订单数据
	if userOrders.IsSnapshot {
		synchronized := false
		for _, s := range userStrategyList {
			adapter, err := helper.NewExchangeAdapterFromStrategy(engine.svcCtx, s.Get())
			if err != nil {
				logger.Errorf("[StrategyEngine] 创建交易所适配器失败, account: %s, exchange: %s, %v",
					s.Get().Account, s.Get().Exchange, err)
				continue
			}

			err = adapter.SyncInactiveOrders(engine.ctx)
			if err != nil {
				logger.Errorf("[StrategyEngine] 同步非活跃订单失败, account: %s, %v", userOrders.Account, err)
				continue
			}

			synchronized = true
			break
		}

		if !synchronized {
			logger.Errorf("[StrategyEngine] 无法完成同步非活跃订单, account: %s", userOrders.Account)
			return errors.New("synchronization failed")
		}
	}

	// 更新活跃订单
	err := util.Tx(engine.ctx, engine.svcCtx.DbClient, func(tx *ent.Tx) error {
		for _, item := range newOrders {
			err := engine.svcCtx.OrderModel.Upsert(engine.ctx, item)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		logger.Errorf("[StrategyEngine] 保存用户活跃订单失败, %v", err)
	}
	return err
}

func (engine *StrategyEngine) userStrategyList(account string) []Strategy {
	userStrategyList := make([]Strategy, 0)

	// 查询账户策略
	engine.mutex.Lock()
	userStrategyIds, ok := engine.userStrategyMap[account]
	if !ok {
		engine.mutex.Unlock()
		return nil
	}

	for _, guid := range userStrategyIds {
		s, ok := engine.strategyMap[guid]
		if ok {
			userStrategyList = append(userStrategyList, s)
		}
	}

	engine.mutex.Unlock()

	return userStrategyList
}

func (engine *StrategyEngine) parseLightOrders(account string, orders []*exchange.Order) ([]ent.Order, error) {
	newOrders := make([]ent.Order, 0)
	for _, ord := range orders {
		marketIndex, err := strconv.ParseInt(ord.Symbol, 10, 64)
		if err != nil {
			logger.Fatalf("[StrategyEngine] 解析市场ID失败, account: %s, symbol: %s", account, ord.Symbol)
			return nil, err
		}

		symbol, err := engine.svcCtx.LighterCache.GetSymbolByMarketId(engine.ctx, uint8(marketIndex))
		if err != nil {
			logger.Fatalf("[StrategyEngine] 查询市场代币符号失败, account: %s, marketIndex: %d", account, marketIndex)
			return nil, err
		}

		args := ent.Order{
			Exchange:          exchange.Lighter,
			Account:           account,
			Symbol:            symbol,
			OrderID:           ord.OrderID,
			ClientOrderID:     ord.ClientOrderID,
			BaseAmount:        ord.BaseAmount,
			FilledBaseAmount:  ord.FilledBaseAmount,
			FilledQuoteAmount: ord.FilledQuoteAmount,
			Status:            ord.Status,
			Timestamp:         ord.Timestamp,
		}
		newOrders = append(newOrders, args)
	}

	return newOrders, nil
}

func (engine *StrategyEngine) handleUserOrders(userOrders exchange.UserOrders, newOrders []ent.Order) error {
	logger.Debugf("[StrategyEngine] 同步用户订单开始, account: %s", userOrders.Account)
	err := engine.syncUserOrders(userOrders, newOrders)
	if err != nil {
		logger.Errorf("[StrategyEngine] 同步用户订单失败, account: %s, %v", userOrders.Account, err)
		return err
	}
	logger.Debugf("[StrategyEngine] 同步用户订单结束, account: %s", userOrders.Account)

	userStrategyList := engine.userStrategyList(userOrders.Account)
	for _, item := range userStrategyList {
		s := item.Get()

		logger.Debugf("[StrategyEngine] 执行用户策略开始, id: %s, account: %s, symbol: %s", s.GUID, s.Account, s.Symbol)
		if err = item.OnUpdate(engine.ctx); err != nil {
			logger.Errorf("[StrategyEngine] 执行用户策略失败, id: %s, account: %s, symbol: %s, %v", s.GUID, s.Account, s.Symbol, err)
		}
		logger.Debugf("[StrategyEngine] 执行用户策略结束, id: %s, account: %s, symbol: %s", s.GUID, s.Account, s.Symbol)
	}

	return nil
}
