package strategy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/order"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/strategy"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/helper"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/model"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/samber/lo"
)

type GridStrategyState struct {
	ctx         context.Context
	svcCtx      *svc.ServiceContext
	strategy    *ent.Strategy
	account     helper.AmbiguousAccount
	sortedGrids []*ent.Grid
	orders      map[int64]*ent.Order
}

func getUpperLevel(sortedGrids []*ent.Grid, level int) *ent.Grid {
	idx := len(sortedGrids)
	for i, item := range sortedGrids {
		if item.Level == level {
			idx = i
			break
		}
	}

	if idx < len(sortedGrids)-1 {
		return sortedGrids[idx+1]
	}
	return nil
}

func getLowerLevel(sortedGrids []*ent.Grid, level int) *ent.Grid {
	idx := -1
	for i, item := range sortedGrids {
		if item.Level == level {
			idx = i
			break
		}
	}

	if idx > 0 {
		return sortedGrids[idx-1]
	}
	return nil
}

func LoadGridStrategyState(ctx context.Context, svcCtx *svc.ServiceContext, s *ent.Strategy) (*GridStrategyState, error) {
	// 初始交易账户
	account := helper.AmbiguousAccount{}
	switch s.Exchange {
	case exchange.Lighter:
		signer, err := helper.GetLighterClient(svcCtx, s)
		if err != nil {
			return nil, err
		}
		account.Signer = signer
	default:
		return nil, errors.New("exchange unsupported")
	}

	// 查询网格列表
	sortedGrids, err := svcCtx.GridModel.FindAllByStrategyIdOrderAsc(ctx, s.GUID)
	if err != nil {
		return nil, err
	}

	// 查询关联订单
	clientOrderIds := make([]int64, 0, len(sortedGrids)*2)
	for _, item := range sortedGrids {
		if item.BuyClientOrderId != nil {
			clientOrderIds = append(clientOrderIds, *item.BuyClientOrderId)
		}
		if item.SellClientOrderId != nil {
			clientOrderIds = append(clientOrderIds, *item.SellClientOrderId)
		}
	}
	orders, err := svcCtx.OrderModel.FindAllByAccountClientOrderIds(ctx, s.Exchange, s.Account, clientOrderIds)
	if err != nil {
		return nil, err
	}

	state := &GridStrategyState{
		ctx:         ctx,
		svcCtx:      svcCtx,
		strategy:    s,
		account:     account,
		sortedGrids: sortedGrids,
		orders:      make(map[int64]*ent.Order),
	}
	for _, item := range orders {
		state.orders[item.ClientOrderId] = item
	}

	return state, nil
}

func (state *GridStrategyState) Rebalance() error {
	for idx := range state.sortedGrids {
		err := state.checkAndRebalanceLevel(idx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (state *GridStrategyState) sendOrderFilledNotice(level *ent.Grid, ord *ent.Order) {
	text := fmt.Sprintf("🚨%s %s %s 订单成交通知\n\n", state.strategy.Exchange, state.strategy.Symbol, state.strategy.Mode)
	text += fmt.Sprintf("📊 方向: %s\n", lo.If(ord.Side == order.SideBuy, "买入").Else("卖出"))
	text += fmt.Sprintf("🔗 网格编号: #%d 📋\n", level.Level)
	text += fmt.Sprintf("🔢 成交数量: %s\n", ord.FilledBaseAmount)
	text += fmt.Sprintf("💥 成交价格: %s USD\n", ord.Price)
	text += fmt.Sprintf("⏰ 成交时间: %s\n", util.FormaTime(time.Unix(ord.Timestamp, 0)))

	chatId := util.ChatId(state.strategy.Owner)
	_, err := util.SendMarkdownMessage(state.svcCtx.Bot, chatId, text, nil)
	if err != nil {
		logger.Debugf("[GridStrategyState] 发送订单成交通知失败, chat: %d, %v", chatId, err)
	}
}

func (state *GridStrategyState) sendGridMatchedNotice(level *ent.Grid, trade *ent.MatchedTrade) {
	// 🚨Lighter SOL short 网格配对通知

	// 做空数量:
	// 做空价格:
	// 平空数量:
	// 平空价格:
	// 实现利润:
	// 配对时间: 11/6/2025 14:14:50

	// 🚨Lighter SOL long 网格配对通知

	// 做多数量:
	// 做多价格:
	// 平多数量:
	// 平多价格:
	// 实现利润:
	// 配对时间: 11/6/2025 14:14:50

}

func (state *GridStrategyState) checkAndRebalanceLevel(idx int) error {
	// 查询关联订单
	var buyOrder *ent.Order
	var sellOrder *ent.Order
	level := state.sortedGrids[idx]

	var ok bool
	if level.BuyClientOrderId != nil {
		buyOrder, ok = state.orders[*level.BuyClientOrderId]
		if !ok {
			buyOrder = nil
		}
	}
	if level.SellClientOrderId != nil {
		sellOrder, ok = state.orders[*level.SellClientOrderId]
		if !ok {
			sellOrder = nil
		}
	}

	// 处理买入订单
	adapter := helper.NewExchangeAdapter(state.svcCtx, state.account)
	if buyOrder != nil && buyOrder.Status == order.StatusFilled {
		logger.Infof("[%s %s] #%d 买单成交, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, level.Level, buyOrder.Price, buyOrder.FilledBaseAmount)

		newCreated, err := state.svcCtx.MatchedTradeModel.EnsureBuyOrder(state.ctx, state.strategy.GUID, buyOrder)
		if err != nil {
			logger.Errorf("[GridStrategyState] 保存匹配记录失败, strategy: %s, buyClientOrderId: %d, %v", state.strategy.GUID, buyOrder.ClientOrderId, err)
			return err
		}

		if newCreated {
			state.sendOrderFilledNotice(level, buyOrder)
		}

		upperLevel := getUpperLevel(state.sortedGrids, level.Level)
		if upperLevel != nil {
			if upperLevel.SellClientOrderId == nil {
				quantity := buyOrder.FilledBaseAmount
				if state.strategy.Mode == strategy.ModeShort {
					quantity = upperLevel.Quantity
				}

				sellOrderId, err := adapter.CreateLimitOrder(state.ctx, state.strategy.Symbol, true, false, upperLevel.Price, quantity)
				if err != nil {
					logger.Errorf("[%s %s] #%d 下单卖单错误, 价格: %s, 数量: %s, %v",
						state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, upperLevel.Price, quantity, err)
					return err
				}

				logger.Infof("[%s %s] #%d 下单卖单, sellOrderId: %d, 价格: %s, 数量: %s",
					state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, sellOrderId, upperLevel.Price, quantity)

				// 更新数据状态
				err = util.Tx(state.ctx, state.svcCtx.DbClient, func(tx *ent.Tx) error {
					m := model.NewGridModel(tx.Grid)
					err = m.UpdateBuyClientOrderId(state.ctx, level.ID, nil)
					if err != nil {
						return err
					}

					err = m.UpdateSellClientOrderId(state.ctx, upperLevel.ID, &sellOrderId)
					if err != nil {
						return err
					}

					return model.NewMatchedTradeModel(tx.MatchedTrade).UpdateByBuyOrder(
						state.ctx, state.strategy.GUID, buyOrder, sellOrderId, &quantity, nil, nil)
				})
				if err != nil {
					logger.Errorf("[GridStrategyState] 更新网格状态失败, level: %d, buyClientOrderId: nil, upperLevel: %d, sellClientOrderId: %d, %v",
						level.ID, upperLevel.ID, sellOrderId, err)
				} else {
					level.BuyClientOrderId = nil
					upperLevel.SellClientOrderId = &sellOrderId
				}
			} else {
				logger.Warnf("[%s %s] #%d 取消下单卖单, 价格: %s, 数量: %s, sellClientOrderId: %d",
					state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, upperLevel.Price, upperLevel.Quantity, *upperLevel.SellClientOrderId)
			}
		}
	}

	// 处理卖出订单
	if sellOrder != nil && sellOrder.Status == order.StatusFilled {
		logger.Infof("[%s %s] #%d 卖单成交, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, level.Level, sellOrder.Price, sellOrder.FilledBaseAmount)

		newCreated, err := state.svcCtx.MatchedTradeModel.EnsureSellOrder(state.ctx, state.strategy.GUID, sellOrder)
		if err != nil {
			logger.Errorf("[GridStrategyState] 保存匹配记录失败, strategy: %s, sellClientOrderId: %d, %v", state.strategy.GUID, sellOrder.ClientOrderId, err)
			return err
		}

		if newCreated {
			state.sendOrderFilledNotice(level, sellOrder)
		}

		lowerLevel := getLowerLevel(state.sortedGrids, level.Level)
		if lowerLevel != nil {
			if lowerLevel.BuyClientOrderId == nil {
				quantity := sellOrder.FilledBaseAmount
				if state.strategy.Mode == strategy.ModeLong {
					quantity = lowerLevel.Quantity
				}

				buyOrderId, err := adapter.CreateLimitOrder(state.ctx, state.strategy.Symbol, false, false, lowerLevel.Price, quantity)
				if err != nil {
					logger.Errorf("[%s %s] #%d 下单买单错误, 价格: %s, 数量: %s, %v",
						state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, lowerLevel.Price, quantity, err)
					return err
				}

				logger.Infof("[%s %s] #%d 下单买单, buyOrderId: %d, 价格: %s, 数量: %s",
					state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, buyOrderId, lowerLevel.Price, quantity)

				// 更新数据状态
				err = util.Tx(state.ctx, state.svcCtx.DbClient, func(tx *ent.Tx) error {
					m := model.NewGridModel(tx.Grid)
					err = m.UpdateSellClientOrderId(state.ctx, level.ID, nil)
					if err != nil {
						return err
					}

					err = m.UpdateBuyClientOrderId(state.ctx, lowerLevel.ID, &buyOrderId)
					if err != nil {
						return err
					}

					return model.NewMatchedTradeModel(tx.MatchedTrade).UpdateBySellOrder(
						state.ctx, state.strategy.GUID, sellOrder, buyOrderId, &quantity, nil, nil)
				})
				if err != nil {
					logger.Errorf("[GridStrategyState] 更新网格状态失败, level: %d, sellClientOrderId: nil, lowerLevel: %d, buyClientOrderId: %d, %v",
						level.ID, lowerLevel.ID, buyOrderId, err)
				} else {
					level.BuyClientOrderId = nil
					lowerLevel.SellClientOrderId = &buyOrderId
				}
			} else {
				logger.Infof("[%s %s] #%d 取消下单买单, 价格: %s, 数量: %s, buyClientOrderId: %d",
					state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, lowerLevel.Price, lowerLevel.Quantity, *lowerLevel.BuyClientOrderId)
			}
		}
	}

	return nil
}
