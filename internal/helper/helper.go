package helper

import (
	"context"
	"errors"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	entstrategy "github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

// DefaultSlippageBps 默认滑点容忍度(基点)
const (
	DefaultSlippageBps = 50
)

// MarketMetadata 市场元数据
// 包含交易所支持的最小区块数量、价格精度等信息
type MarketMetadata struct {
	MinBaseAmount          decimal.Decimal // 最小基础货币数量
	MinQuoteAmount         decimal.Decimal // 最小计价货币数量
	SupportedSizeDecimals  uint8           // 支持的数量小数位数
	SupportedPriceDecimals uint8           // 支持的价格小数位数
	SupportedQuoteDecimals uint8           // 支持的计价小数位数
}

// StrategyEngine 策略引擎接口
type StrategyEngine interface {
	StopStrategy(id string)
}

// GetAccountInfo 获取账户信息
// 根据策略记录中的交易所类型，返回对应的账户信息
// ctx 上下文，svcCtx 服务上下文，record 策略记录
// 返回值: 账户信息，错误信息
func GetAccountInfo(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) (*exchange.Account, error) {
	switch record.Exchange {
	case exchange.Lighter:
		return GetLighterAccountInfo(ctx, svcCtx, record.Account)
	case exchange.Paradex:
		return GetParadexAccountInfo(ctx, svcCtx, record)
	case exchange.Variational:
		return GetVariationalAccountInfo(ctx, svcCtx, record)
	default:
		return nil, errors.New("exchange unsupported")
	}
}

// GetMarketMetadata 获取市场元数据
// 返回指定交易所和交易对的市场配置信息(最小订单数量、价格精度等)
func GetMarketMetadata(ctx context.Context, svcCtx *svc.ServiceContext, exchangeType, symbol string) (MarketMetadata, error) {
	switch exchangeType {
	case exchange.Lighter:
		metadata, err := svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
		if err != nil {
			return MarketMetadata{}, err
		}

		ret := MarketMetadata{
			MinBaseAmount:          metadata.MinBaseAmount,
			MinQuoteAmount:         metadata.MinQuoteAmount,
			SupportedSizeDecimals:  metadata.SupportedSizeDecimals,
			SupportedPriceDecimals: metadata.SupportedPriceDecimals,
			SupportedQuoteDecimals: metadata.SupportedQuoteDecimals,
		}
		return ret, nil
	case exchange.Paradex:
		metadata, err := svcCtx.ParadexCache.GetMarketMetadata(ctx, paradex.FormatUsdPerpMarket(symbol))
		if err != nil {
			return MarketMetadata{}, err
		}

		ret := MarketMetadata{
			MinBaseAmount:          metadata.OrderSizeIncrement,
			MinQuoteAmount:         metadata.MinNotional,
			SupportedSizeDecimals:  uint8(-metadata.OrderSizeIncrement.Exponent()),
			SupportedPriceDecimals: uint8(-metadata.PriceTickSize.Exponent()),
			SupportedQuoteDecimals: uint8(-metadata.PriceTickSize.Exponent()),
		}
		return ret, nil
	case exchange.Variational:
		quote, err := svcCtx.VariationalClient.SimpleQuote(ctx, symbol, decimal.NewFromFloat(0.0001))
		if err != nil {
			return MarketMetadata{}, err
		}

		minBaseAmount := quote.QtyLimits.Ask.MinQty
		if minBaseAmount.LessThan(quote.QtyLimits.Bid.MinQty) {
			minBaseAmount = quote.QtyLimits.Bid.MinQty
		}

		askPriceDecimals := uint8(-quote.Ask.Exponent())
		bidPriceDecimals := uint8(-quote.Bid.Exponent())
		supportedPriceDecimals := lo.Max([]uint8{askPriceDecimals, bidPriceDecimals})

		ret := MarketMetadata{
			MinBaseAmount:          minBaseAmount,
			MinQuoteAmount:         decimal.Zero,
			SupportedSizeDecimals:  uint8(-minBaseAmount.Exponent()),
			SupportedPriceDecimals: supportedPriceDecimals,
			SupportedQuoteDecimals: supportedPriceDecimals,
		}
		return ret, nil
	default:
		return MarketMetadata{}, errors.New("exchange unsupported")
	}
}

// GetLastTradePrice 获取最新成交价格
func GetLastTradePrice(ctx context.Context, svcCtx *svc.ServiceContext, exchangeType, symbol string) (decimal.Decimal, error) {
	switch exchangeType {
	case exchange.Lighter:
		metadata, err := svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
		if err != nil {
			return decimal.Zero, err
		}

		return svcCtx.LighterClient.GetLastTradePrice(ctx, uint(metadata.MarketID))
	case exchange.Paradex:
		marketSummary, err := svcCtx.ParadexClient.GetMarketSummary(ctx, paradex.FormatUsdPerpMarket(symbol))
		if err != nil {
			return decimal.Zero, err
		}

		if len(marketSummary.Results) == 0 {
			return decimal.Zero, nil
		}

		return marketSummary.Results[0].LastTradedPrice, nil
	case exchange.Variational:
		quote, err := svcCtx.VariationalClient.SimpleQuote(ctx, symbol, decimal.NewFromFloat(0.0001))
		if err != nil {
			return decimal.Zero, err
		}
		return quote.Ask, nil
	default:
		return decimal.Zero, errors.New("exchange unsupported")
	}
}

// StopStrategyAndCancelOrders 停止策略并取消所有订单
// 1. 停止网格策略运行
// 2. 取消所有挂出的订单
// 3. 删除网格和成交记录
// 4. 更新策略状态为inactive
func StopStrategyAndCancelOrders(ctx context.Context, svcCtx *svc.ServiceContext, strategyEngine StrategyEngine, record *ent.Strategy) error {
	// 停止网格策略
	strategyEngine.StopStrategy(record.GUID)

	// 取消用户订单
	adapter, err := NewExchangeAdapterFromStrategy(svcCtx, record)
	if err != nil {
		return err
	}
	err = adapter.CancalAllOrders(ctx, record.Symbol)
	if err != nil {
		return err
	}

	// 更新策略状态
	return util.Tx(ctx, svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewMatchedTradeModel(tx.MatchedTrade).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatus(ctx, record.ID, entstrategy.StatusInactive)
	})
}

// StopStrategyAndClosePosition 停止策略并平仓
// 在停止策略的基础上 additionally 执行平仓操作
func StopStrategyAndClosePosition(ctx context.Context, svcCtx *svc.ServiceContext, strategyEngine StrategyEngine, record *ent.Strategy) error {
	err := StopStrategyAndCancelOrders(ctx, svcCtx, strategyEngine, record)
	if err != nil {
		return err
	}

	adapter, err := NewExchangeAdapterFromStrategy(svcCtx, record)
	if err != nil {
		return err
	}

	// 获取滑点容忍度
	slippageBps := DefaultSlippageBps
	if record.SlippageBps != nil {
		slippageBps = *record.SlippageBps
	}

	// 根据策略模式确定平仓方向
	side := lo.If(record.Mode == strategy.ModeLong, LONG).Else(SHORT)
	return adapter.ClosePosition(ctx, record.Symbol, side, slippageBps)
}
