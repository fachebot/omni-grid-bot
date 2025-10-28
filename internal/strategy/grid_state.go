package strategy

import (
	"context"
	"errors"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/order"
	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/helper"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/model"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
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
		state.orders[item.ClientOrderID] = item
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
		logger.Infof("[%s %s] #%d 买单成交, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, level.Level, level.Price, level.Quantity)

		upperLevel := getUpperLevel(state.sortedGrids, level.Level)
		if upperLevel != nil {
			if upperLevel.SellClientOrderId == nil {
				sellOrderId, err := adapter.CreateLimitOrder(state.ctx, state.strategy.Symbol, true, false, upperLevel.Price, level.Quantity)
				if err != nil {
					logger.Errorf("[%s %s] #%d 下单卖单错误, 价格: %s, 数量: %s, %v",
						state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, upperLevel.Price, level.Quantity, err)
					return err
				}

				logger.Infof("[%s %s] #%d 下单卖单, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, upperLevel.Price, level.Quantity)

				// 更新数据状态
				err = util.Tx(state.ctx, state.svcCtx.DbClient, func(tx *ent.Tx) error {
					m := model.NewGridModel(tx.Grid)
					err = m.UpdateBuyClientOrderId(state.ctx, level.ID, nil)
					if err != nil {
						return err
					}

					return m.UpdateSellClientOrderId(state.ctx, upperLevel.ID, &sellOrderId)
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
					state.strategy.Symbol, state.strategy.Mode, upperLevel.Level, upperLevel.Price, level.Quantity, *upperLevel.SellClientOrderId)
			}
		}
	}

	// 处理卖出订单
	if sellOrder != nil && sellOrder.Status == order.StatusFilled {
		logger.Infof("[%s %s] #%d 卖单成交, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, level.Level, level.Price, level.Quantity)

		lowerLevel := getLowerLevel(state.sortedGrids, level.Level)
		if lowerLevel != nil {
			if lowerLevel.BuyClientOrderId == nil {
				buyOrderId, err := adapter.CreateLimitOrder(state.ctx, state.strategy.Symbol, false, false, lowerLevel.Price, level.Quantity)
				if err != nil {
					logger.Errorf("[%s %s] #%d 下单买单错误, 价格: %s, 数量: %s, %v",
						state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, lowerLevel.Price, level.Quantity, err)
					return err
				}

				logger.Infof("[%s %s] #%d 下单买单, 价格: %s, 数量: %s", state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, lowerLevel.Price, level.Quantity)

				// 更新数据状态
				err = util.Tx(state.ctx, state.svcCtx.DbClient, func(tx *ent.Tx) error {
					m := model.NewGridModel(tx.Grid)
					err = m.UpdateSellClientOrderId(state.ctx, level.ID, nil)
					if err != nil {
						return err
					}

					return m.UpdateBuyClientOrderId(state.ctx, lowerLevel.ID, &buyOrderId)
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
					state.strategy.Symbol, state.strategy.Mode, lowerLevel.Level, lowerLevel.Price, level.Quantity, *lowerLevel.BuyClientOrderId)
			}
		}
	}

	return nil
}
