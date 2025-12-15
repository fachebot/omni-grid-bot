package strategy

import (
	"context"
	"fmt"
	"strings"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/fachebot/omni-grid-bot/internal/util/format"
	"github.com/shopspring/decimal"
)

type StrategyEngine interface {
	StopStrategy(id string)
}

type GridStrategy struct {
	engine   StrategyEngine
	svcCtx   *svc.ServiceContext
	strategy *ent.Strategy
}

func NewGridStrategy(svcCtx *svc.ServiceContext, engine StrategyEngine, s *ent.Strategy) *GridStrategy {
	return &GridStrategy{svcCtx: svcCtx, engine: engine, strategy: s}
}

func (s *GridStrategy) Get() *ent.Strategy {
	return s.strategy
}

func (s *GridStrategy) Update(entStrategy *ent.Strategy) {
	s.strategy = entStrategy
}

func (s *GridStrategy) OnTicker(ctx context.Context, price decimal.Decimal) {
	logger.Tracef("[GridStrategy] 收到行情更新, id: %s, symbol: %s, account: %s, price: %s",
		s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, price.String())

	switch s.strategy.Mode {
	case strategy.ModeLong:
		if s.strategy.TriggerStopLossPrice != nil &&
			s.strategy.TriggerStopLossPrice.GreaterThan(decimal.Zero) &&
			price.LessThanOrEqual(*s.strategy.TriggerStopLossPrice) {
			s.handleTriggerStopLossPrice(ctx, price, *s.strategy.TriggerStopLossPrice)
			return
		}

		if s.strategy.TriggerTakeProfitPrice != nil &&
			s.strategy.TriggerTakeProfitPrice.GreaterThan(decimal.Zero) &&
			price.GreaterThanOrEqual(*s.strategy.TriggerTakeProfitPrice) {
			s.handleTriggerTakeProfitPrice(ctx, price, *s.strategy.TriggerTakeProfitPrice)
			return
		}
	case strategy.ModeShort:
		if s.strategy.TriggerStopLossPrice != nil &&
			s.strategy.TriggerStopLossPrice.GreaterThan(decimal.Zero) &&
			price.GreaterThanOrEqual(*s.strategy.TriggerStopLossPrice) {
			s.handleTriggerStopLossPrice(ctx, price, *s.strategy.TriggerStopLossPrice)
			return
		}

		if s.strategy.TriggerTakeProfitPrice != nil &&
			s.strategy.TriggerTakeProfitPrice.GreaterThan(decimal.Zero) &&
			price.LessThanOrEqual(*s.strategy.TriggerTakeProfitPrice) {
			s.handleTriggerTakeProfitPrice(ctx, price, *s.strategy.TriggerTakeProfitPrice)
			return
		}
	}
}

func (s *GridStrategy) OnOrdersChanged(ctx context.Context) error {
	state, err := LoadGridStrategyState(ctx, s.svcCtx, s.strategy)
	if err != nil {
		logger.Errorf("[GridStrategy] 加载策略状态失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return err
	}

	if err = state.Rebalance(); err != nil {
		logger.Errorf("[GridStrategy] 处理网格再平衡失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return err
	}

	return nil
}

func (s *GridStrategy) handleTriggerStopLossPrice(ctx context.Context, price, stopLossPrice decimal.Decimal) {
	logger.Infof("[GridStrategy] 触发止损价格, 停止策略, id: %s, symbol: %s, account: %s, price: %s",
		s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, price.String())

	err := helper.StopStrategyAndClosePosition(ctx, s.svcCtx, s.engine, s.strategy)
	if err != nil {
		logger.Errorf("[GridStrategy] 关闭仓位失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return
	}

	// 发送通知消息
	chatId := util.ChatId(s.strategy.Owner)
	name := util.StrategyName(s.strategy)
	link := fmt.Sprintf("[%s](https://t.me/%s?start=%s)",
		name, s.svcCtx.Bot.Me.Username, s.strategy.GUID)
	text := fmt.Sprintf("📉 **%s %s** 触发止损价格 %s\n\n",
		s.strategy.Symbol, strings.ToUpper(string(s.strategy.Mode)), link)
	text += fmt.Sprintf("💵 当前价格: %s\n", format.Price(price, 5))
	text += fmt.Sprintf("🔔 触发价格: %s\n", format.Price(stopLossPrice, 5))
	text += "\n策略已自动停止并平仓。由于市价滑点问题，可能存在平仓失败的情况，请注意检查仓位是否正常关闭。"
	_, err = util.SendMarkdownMessage(s.svcCtx.Bot, chatId, text, nil)
	if err != nil {
		logger.Debugf("[GridStrategy] 发送触发止损价格通知失败, chat: %d, %v", chatId, err)
	}
}

func (s *GridStrategy) handleTriggerTakeProfitPrice(ctx context.Context, price, takeProfitPrice decimal.Decimal) {
	logger.Infof("[GridStrategy] 触发止盈价格, 停止策略, id: %s, symbol: %s, account: %s, price: %s",
		s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, price.String())

	err := helper.StopStrategyAndClosePosition(ctx, s.svcCtx, s.engine, s.strategy)
	if err != nil {
		logger.Errorf("[GridStrategy] 关闭仓位失败, id: %s, symbol: %s, account: %s, %v",
			s.strategy.GUID, s.strategy.Symbol, s.strategy.Account, err)
		return
	}

	// 发送通知消息
	chatId := util.ChatId(s.strategy.Owner)
	name := util.StrategyName(s.strategy)
	link := fmt.Sprintf("[%s](https://t.me/%s?start=%s)",
		name, s.svcCtx.Bot.Me.Username, s.strategy.GUID)
	text := fmt.Sprintf("📈 **%s %s** 触发止盈价格 %s\n\n",
		s.strategy.Symbol, strings.ToUpper(string(s.strategy.Mode)), link)
	text += fmt.Sprintf("💵 当前价格: %s\n", format.Price(price, 5))
	text += fmt.Sprintf("🔔 触发价格: %s\n", format.Price(takeProfitPrice, 5))
	text += "\n策略已自动停止并平仓。由于市价滑点问题，可能存在平仓失败的情况，请注意检查仓位是否正常关闭。"
	_, err = util.SendMarkdownMessage(s.svcCtx.Bot, chatId, text, nil)
	if err != nil {
		logger.Debugf("[GridStrategy] 发送触发止盈价格通知失败, chat: %d, %v", chatId, err)
	}
}
