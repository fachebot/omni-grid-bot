package engine

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	gridstrategy "github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/util"
)

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

// handleOrderCancelled 处理订单被取消的情况
func (engine *StrategyEngine) handleOrderCancelled(record *ent.Strategy) {
	// 停止网格策略
	err := helper.StopStrategyAndCancelOrders(engine.ctx, engine.svcCtx, engine, record)
	if err != nil {
		logger.Warnf("[StrategyEngine] 停止策略并取消订单失败, exchange: %s, account: %s, symbol: %s, side: %s, %v",
			record.Exchange, record.Account, record.Symbol, record.Mode, err)
	}

	// 发送通知消息
	chatId := util.ChatId(record.Owner)
	name := util.StrategyName(record)
	link := fmt.Sprintf("[%s](https://t.me/%s?start=%s)",
		name, engine.svcCtx.Bot.Me.Username, record.GUID)
	text := fmt.Sprintf("🚨 **%s %s** 策略已停止 %s\n\n",
		record.Symbol, strings.ToUpper(string(record.Mode)), link)
	text += "由于订单被意外取消，策略已自动停止，请手动关闭仓位。\n\n**注意**：`策略运行中请勿手动进行操作，以免干扰策略正常运行。`"
	_, err = util.SendMarkdownMessage(engine.svcCtx.Bot, chatId, text, nil)
	if err != nil {
		logger.Debugf("[StrategyEngine] 发送策略已停止通知失败, chat: %d, %v", chatId, err)
	}
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
		if errors.Is(err, gridstrategy.ErrOrderCanceled) {
			engine.handleOrderCancelled(s)
		}

		logger.Errorf("[StrategyEngine] 执行用户策略失败, id: %s, account: %s, symbol: %s, %v",
			s.GUID, s.Account, s.Symbol, err)
		return
	}

	engine.removeFromRetryQueue(s.GUID)

	logger.Debugf("[StrategyEngine] 执行用户策略结束, id: %s, account: %s, symbol: %s",
		s.GUID, s.Account, s.Symbol)
}
