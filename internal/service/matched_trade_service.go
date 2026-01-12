package service

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/model"
)

// MatchedTradeService 处理匹配交易的业务逻辑
type MatchedTradeService struct {
	model *model.MatchedTradeModel
}

// NewMatchedTradeService 创建匹配交易服务实例
func NewMatchedTradeService(model *model.MatchedTradeModel) *MatchedTradeService {
	return &MatchedTradeService{model: model}
}

// RecordAndMatchBuyOrder 记录并匹配买入订单
// 业务逻辑：判断是否是首次记录、判断是否完成配对
// 返回: isFirstRecord(是否首次记录), completedPair(完成的配对), error
func (s *MatchedTradeService) RecordAndMatchBuyOrder(
	ctx context.Context, strategy *ent.Strategy, buyOrder *ent.Order,
) (isFirstRecord bool, completedPair *ent.MatchedTrade, err error) {
	// 查询是否已存在记录
	existing, err := s.model.FindByBuyClientOrderId(ctx, strategy.GUID, buyOrder.ClientOrderId)
	if err != nil && !ent.IsNotFound(err) {
		return false, nil, err
	}

	// 如果记录已存在，更新买入订单信息
	if err == nil && existing != nil {
		// 数据存储：更新买入订单信息
		err = s.model.UpdateBuyOrderInfo(ctx, strategy.GUID, buyOrder.ClientOrderId,
			buyOrder.FilledBaseAmount, buyOrder.FilledQuoteAmount, buyOrder.Timestamp)
		if err != nil {
			return false, nil, err
		}

		// 业务逻辑：判断是否是首次记录（更新前买入订单时间戳为空）
		isFirstRecord = existing.BuyOrderTimestamp == nil

		// 业务逻辑：判断是否完成配对（卖出订单已存在且已成交）
		if existing.SellClientOrderId != nil && existing.SellOrderTimestamp != nil {
			// 创建返回对象，包含更新后的买入订单信息
			completedPair = existing
			completedPair.BuyBaseAmount = &buyOrder.FilledBaseAmount
			completedPair.BuyQuoteAmount = &buyOrder.FilledQuoteAmount
			completedPair.BuyOrderTimestamp = &buyOrder.Timestamp
		}

		return isFirstRecord, completedPair, nil
	}

	// 如果记录不存在，创建新记录
	isFirstRecord = true
	args := ent.MatchedTrade{
		StrategyId:        strategy.GUID,
		Account:           strategy.Account,
		Symbol:            buyOrder.Symbol,
		BuyClientOrderId:  &buyOrder.ClientOrderId,
		BuyBaseAmount:     &buyOrder.FilledBaseAmount,
		BuyQuoteAmount:    &buyOrder.FilledQuoteAmount,
		BuyOrderTimestamp: &buyOrder.Timestamp,
	}
	// 数据存储：创建新记录
	if err = s.model.Create(ctx, args); err != nil {
		return isFirstRecord, nil, err
	}

	return isFirstRecord, nil, nil
}

// RecordAndMatchSellOrder 记录并匹配卖出订单
// 业务逻辑：判断是否是首次记录、判断是否完成配对
// 返回: isFirstRecord(是否首次记录), completedPair(完成的配对), error
func (s *MatchedTradeService) RecordAndMatchSellOrder(
	ctx context.Context, strategy *ent.Strategy, sellOrder *ent.Order,
) (isFirstRecord bool, completedPair *ent.MatchedTrade, err error) {
	// 查询是否已存在记录
	existing, err := s.model.FindBySellClientOrderId(ctx, strategy.GUID, sellOrder.ClientOrderId)
	if err != nil && !ent.IsNotFound(err) {
		return false, nil, err
	}

	// 如果记录已存在，更新卖出订单信息
	if err == nil && existing != nil {
		// 数据存储：更新卖出订单信息
		err = s.model.UpdateSellOrderInfo(ctx, strategy.GUID, sellOrder.ClientOrderId,
			sellOrder.FilledBaseAmount, sellOrder.FilledQuoteAmount, sellOrder.Timestamp)
		if err != nil {
			return false, nil, err
		}

		// 业务逻辑：判断是否是首次记录（更新前卖出订单时间戳为空）
		isFirstRecord = existing.SellOrderTimestamp == nil

		// 业务逻辑：判断是否完成配对（买入订单已存在且已成交）
		if existing.BuyClientOrderId != nil && existing.BuyOrderTimestamp != nil {
			// 创建返回对象，包含更新后的卖出订单信息
			completedPair = existing
			completedPair.SellBaseAmount = &sellOrder.FilledBaseAmount
			completedPair.SellQuoteAmount = &sellOrder.FilledQuoteAmount
			completedPair.SellOrderTimestamp = &sellOrder.Timestamp
		}

		return isFirstRecord, completedPair, nil
	}

	// 如果记录不存在，创建新记录
	isFirstRecord = true
	args := ent.MatchedTrade{
		StrategyId:         strategy.GUID,
		Account:            strategy.Account,
		Symbol:             sellOrder.Symbol,
		SellClientOrderId:  &sellOrder.ClientOrderId,
		SellBaseAmount:     &sellOrder.FilledBaseAmount,
		SellQuoteAmount:    &sellOrder.FilledQuoteAmount,
		SellOrderTimestamp: &sellOrder.Timestamp,
	}
	// 数据存储：创建新记录
	if err = s.model.Create(ctx, args); err != nil {
		return isFirstRecord, nil, err
	}

	return isFirstRecord, nil, nil
}
