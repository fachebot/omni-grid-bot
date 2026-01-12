package model

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/matchedtrade"
	"github.com/shopspring/decimal"
)

type MatchedTradeModel struct {
	client *ent.MatchedTradeClient
}

func NewMatchedTradeModel(client *ent.MatchedTradeClient) *MatchedTradeModel {
	return &MatchedTradeModel{client: client}
}

func (m *MatchedTradeModel) Create(ctx context.Context, args ent.MatchedTrade) error {
	return m.client.Create().
		SetStrategyId(args.StrategyId).
		SetAccount(args.Account).
		SetSymbol(args.Symbol).
		SetNillableBuyClientOrderId(args.BuyClientOrderId).
		SetNillableBuyBaseAmount(args.BuyBaseAmount).
		SetNillableBuyQuoteAmount(args.BuyQuoteAmount).
		SetNillableBuyOrderTimestamp(args.BuyOrderTimestamp).
		SetNillableSellClientOrderId(args.SellClientOrderId).
		SetNillableSellBaseAmount(args.SellBaseAmount).
		SetNillableSellQuoteAmount(args.SellQuoteAmount).
		SetNillableSellOrderTimestamp(args.SellOrderTimestamp).
		SetNillableProfit(args.Profit).
		Exec(ctx)
}

func (m *MatchedTradeModel) QueryTotalProfit(ctx context.Context, strategyId string) (decimal.Decimal, error) {
	var v []struct{ Sum decimal.Decimal }
	err := m.client.Query().
		Where(matchedtrade.StrategyIdEQ(strategyId)).
		Aggregate(ent.Sum(matchedtrade.FieldProfit)).
		Scan(ctx, &v)
	if err != nil || len(v) == 0 {
		return decimal.Zero, nil
	}
	return v[0].Sum, nil
}

func (m *MatchedTradeModel) FinAllMatchedTrades(ctx context.Context, strategyId string, offset, limit int) ([]*ent.MatchedTrade, int, error) {
	q := m.client.Query().
		Where(matchedtrade.StrategyIdEQ(strategyId), matchedtrade.ProfitNotNil())
	count, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	r, err := q.Order(matchedtrade.ByUpdateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return r, count, nil
}

func (m *MatchedTradeModel) QueryOpeLongPositionAndCost(ctx context.Context, strategyId string) (position, cost decimal.Decimal, err error) {
	trades, err := m.client.Query().Where(
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.BuyOrderTimestampNotNil(),
		matchedtrade.SellOrderTimestampIsNil(),
	).All(ctx)
	if err != nil {
		return position, cost, err
	}

	for _, item := range trades {
		if item.BuyQuoteAmount == nil || item.BuyBaseAmount == nil {
			continue
		}

		cost = cost.Add(*item.BuyQuoteAmount)
		position = position.Add(*item.BuyBaseAmount)
	}
	return position, cost, nil
}

func (m *MatchedTradeModel) QueryOpenShortPositionAndCost(ctx context.Context, strategyId string) (position, cost decimal.Decimal, err error) {
	trades, err := m.client.Query().Where(
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.SellOrderTimestampNotNil(),
		matchedtrade.BuyOrderTimestampIsNil(),
	).All(ctx)
	if err != nil {
		return position, cost, err
	}

	for _, item := range trades {
		if item.SellQuoteAmount == nil || item.SellBaseAmount == nil {
			continue
		}

		cost = cost.Add(*item.SellQuoteAmount)
		position = position.Add(*item.SellBaseAmount)
	}
	return position, cost, nil
}

// FindByBuyClientOrderId 根据策略ID和买入订单ID查询匹配交易记录
func (m *MatchedTradeModel) FindByBuyClientOrderId(ctx context.Context, strategyId, buyClientOrderId string) (*ent.MatchedTrade, error) {
	return m.client.Query().
		Where(
			matchedtrade.StrategyIdEQ(strategyId),
			matchedtrade.BuyClientOrderIdEQ(buyClientOrderId),
		).
		First(ctx)
}

// FindBySellClientOrderId 根据策略ID和卖出订单ID查询匹配交易记录
func (m *MatchedTradeModel) FindBySellClientOrderId(ctx context.Context, strategyId, sellClientOrderId string) (*ent.MatchedTrade, error) {
	return m.client.Query().
		Where(
			matchedtrade.StrategyIdEQ(strategyId),
			matchedtrade.SellClientOrderIdEQ(sellClientOrderId),
		).
		First(ctx)
}

// UpdateBuyOrderInfo 更新买入订单信息
func (m *MatchedTradeModel) UpdateBuyOrderInfo(
	ctx context.Context,
	strategyId string,
	buyClientOrderId string,
	buyBaseAmount, buyQuoteAmount decimal.Decimal,
	buyOrderTimestamp int64,
) error {
	return m.client.Update().
		Where(
			matchedtrade.StrategyIdEQ(strategyId),
			matchedtrade.BuyClientOrderIdEQ(buyClientOrderId),
		).
		SetBuyBaseAmount(buyBaseAmount).
		SetBuyQuoteAmount(buyQuoteAmount).
		SetBuyOrderTimestamp(buyOrderTimestamp).
		Exec(ctx)
}

// UpdateSellOrderInfo 更新卖出订单信息
func (m *MatchedTradeModel) UpdateSellOrderInfo(
	ctx context.Context,
	strategyId string,
	sellClientOrderId string,
	sellBaseAmount, sellQuoteAmount decimal.Decimal,
	sellOrderTimestamp int64,
) error {
	return m.client.Update().
		Where(
			matchedtrade.StrategyIdEQ(strategyId),
			matchedtrade.SellClientOrderIdEQ(sellClientOrderId),
		).
		SetSellBaseAmount(sellBaseAmount).
		SetSellQuoteAmount(sellQuoteAmount).
		SetSellOrderTimestamp(sellOrderTimestamp).
		Exec(ctx)
}

func (m *MatchedTradeModel) UpdateProfit(ctx context.Context, id int, value float64) error {
	return m.client.UpdateOneID(id).SetProfit(value).Exec(ctx)
}

func (m *MatchedTradeModel) UpdateByBuyOrder(
	ctx context.Context,
	strategyId string,
	buyOrder *ent.Order,
	sellClientOrderId string,
	sellBaseAmount,
	sellQuoteAmount *decimal.Decimal,
	sellOrderTimestamp *int64) error {

	return m.client.Update().
		Where(matchedtrade.StrategyIdEQ(strategyId), matchedtrade.BuyClientOrderIdEQ(buyOrder.ClientOrderId)).
		SetNillableBuyBaseAmount(&buyOrder.FilledBaseAmount).
		SetNillableBuyQuoteAmount(&buyOrder.FilledQuoteAmount).
		SetNillableBuyOrderTimestamp(&buyOrder.Timestamp).
		SetSellClientOrderId(sellClientOrderId).
		SetNillableSellBaseAmount(sellBaseAmount).
		SetNillableSellQuoteAmount(sellQuoteAmount).
		SetNillableSellOrderTimestamp(sellOrderTimestamp).
		Exec(ctx)
}

func (m *MatchedTradeModel) UpdateBySellOrder(
	ctx context.Context,
	strategyId string,
	sellOrder *ent.Order,
	buyClientOrderId string,
	buyBaseAmount,
	buyQuoteAmount *decimal.Decimal,
	buyOrderTimestamp *int64) error {

	return m.client.Update().
		Where(matchedtrade.StrategyIdEQ(strategyId), matchedtrade.SellClientOrderIdEQ(sellOrder.ClientOrderId)).
		SetNillableSellBaseAmount(&sellOrder.FilledBaseAmount).
		SetNillableSellQuoteAmount(&sellOrder.FilledQuoteAmount).
		SetNillableSellOrderTimestamp(&sellOrder.Timestamp).
		SetBuyClientOrderId(buyClientOrderId).
		SetNillableBuyBaseAmount(buyBaseAmount).
		SetNillableBuyQuoteAmount(buyQuoteAmount).
		SetNillableBuyOrderTimestamp(buyOrderTimestamp).
		Exec(ctx)
}

func (m *MatchedTradeModel) DeleteByStrategyId(ctx context.Context, strategyId string) error {
	_, err := m.client.Delete().Where(matchedtrade.StrategyIdEQ(strategyId)).Exec(ctx)
	return err
}
