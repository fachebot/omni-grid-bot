package model

import (
	"context"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/matchedtrade"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/predicate"
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
	err := m.client.Query().Aggregate(ent.Sum(matchedtrade.FieldProfit)).Scan(ctx, &v)
	if err != nil || len(v) == 0 {
		return decimal.Zero, nil
	}
	return v[0].Sum, nil
}

func (m *MatchedTradeModel) EnsureBuyOrder(ctx context.Context, strategyId string, buyOrder *ent.Order) (matched *ent.MatchedTrade, err error) {
	ps := []predicate.MatchedTrade{
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.BuyClientOrderIdEQ(buyOrder.ClientOrderId),
	}
	r, err := m.client.Query().Where(ps...).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return matched, err
	}

	if err == nil {
		err = m.client.Update().
			Where(ps...).
			SetBuyBaseAmount(buyOrder.FilledBaseAmount).
			SetBuyQuoteAmount(buyOrder.FilledQuoteAmount).
			SetBuyOrderTimestamp(buyOrder.Timestamp).
			Exec(ctx)
		if err != nil {
			return matched, err
		}

		if r.SellClientOrderId != nil && r.SellOrderTimestamp != nil {
			matched = r
			r.BuyBaseAmount = &buyOrder.FilledBaseAmount
			r.BuyQuoteAmount = &buyOrder.FilledQuoteAmount
			r.BuyOrderTimestamp = &buyOrder.Timestamp
		}

		return matched, nil
	}

	args := ent.MatchedTrade{
		StrategyId:        strategyId,
		Symbol:            buyOrder.Symbol,
		BuyClientOrderId:  &buyOrder.ClientOrderId,
		BuyBaseAmount:     &buyOrder.FilledBaseAmount,
		BuyQuoteAmount:    &buyOrder.FilledQuoteAmount,
		BuyOrderTimestamp: &buyOrder.Timestamp,
	}
	if err = m.Create(ctx, args); err != nil {
		return matched, err
	}

	return matched, nil
}

func (m *MatchedTradeModel) EnsureSellOrder(ctx context.Context, strategyId string, sellOrder *ent.Order) (matched *ent.MatchedTrade, err error) {
	ps := []predicate.MatchedTrade{
		matchedtrade.StrategyIdEQ(strategyId),
		matchedtrade.SellClientOrderIdEQ(sellOrder.ClientOrderId),
	}
	r, err := m.client.Query().Where(ps...).First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return matched, err
	}

	if err == nil {
		err = m.client.Update().
			Where(ps...).
			SetSellBaseAmount(sellOrder.FilledBaseAmount).
			SetSellQuoteAmount(sellOrder.FilledQuoteAmount).
			SetSellOrderTimestamp(sellOrder.Timestamp).
			Exec(ctx)

		if err != nil {
			return matched, err
		}

		if r.BuyClientOrderId != nil && r.BuyOrderTimestamp != nil {
			matched = r
			r.SellBaseAmount = &sellOrder.FilledBaseAmount
			r.SellQuoteAmount = &sellOrder.FilledQuoteAmount
			r.SellOrderTimestamp = &sellOrder.Timestamp
		}

		return matched, nil
	}

	args := ent.MatchedTrade{
		StrategyId:         strategyId,
		Symbol:             sellOrder.Symbol,
		SellClientOrderId:  &sellOrder.ClientOrderId,
		SellBaseAmount:     &sellOrder.FilledBaseAmount,
		SellQuoteAmount:    &sellOrder.FilledQuoteAmount,
		SellOrderTimestamp: &sellOrder.Timestamp,
	}
	if err = m.Create(ctx, args); err != nil {
		return matched, err
	}

	return matched, nil
}

func (m *MatchedTradeModel) UpdateProfit(ctx context.Context, id int, value float64) error {
	return m.client.UpdateOneID(id).SetProfit(value).Exec(ctx)
}

func (m *MatchedTradeModel) UpdateByBuyOrder(
	ctx context.Context,
	strategyId string,
	buyOrder *ent.Order,
	sellClientOrderId int64,
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
	buyClientOrderId int64,
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
