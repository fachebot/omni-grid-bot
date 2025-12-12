package engine

import (
	"errors"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
)

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

// subscribeMarketStatus 订阅市场状态
func (engine *StrategyEngine) subscribeMarketStatus(record *ent.Strategy) error {
	switch record.Exchange {
	case exchange.Lighter:
		return engine.lighterSubscriber.SubscribeMarketStats(record.Symbol)
	case exchange.Paradex:
		return engine.paradexSubscriber.SubscribeMarketStats(record.Symbol)
	case exchange.Variational:
		return engine.variationalSubscriber.SubscribeMarketStats(record.Symbol)
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
