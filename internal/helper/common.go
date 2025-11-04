package helper

import (
	"context"
	"errors"

	"github.com/fachebot/perp-dex-grid-bot/internal/exchange"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/shopspring/decimal"
)

type MarketMetadata struct {
	MinBaseAmount          decimal.Decimal // 最小基础货币数量
	MinQuoteAmount         decimal.Decimal // 最小计价货币数量
	OrderQuoteLimit        string          // 订单计价限制（可选字段）
	SupportedSizeDecimals  uint8           // 支持的数量小数位数
	SupportedPriceDecimals uint8           // 支持的价格小数位数
	SupportedQuoteDecimals uint8           // 支持的计价小数位数
}

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
			OrderQuoteLimit:        metadata.OrderQuoteLimit,
			SupportedSizeDecimals:  metadata.SupportedSizeDecimals,
			SupportedPriceDecimals: metadata.SupportedPriceDecimals,
			SupportedQuoteDecimals: metadata.SupportedQuoteDecimals,
		}
		return ret, nil
	default:
		return MarketMetadata{}, errors.New("exchange unsupported")
	}
}

func GetLastTradePrice(ctx context.Context, svcCtx *svc.ServiceContext, exchangeType, symbol string) (decimal.Decimal, error) {
	switch exchangeType {
	case exchange.Lighter:
		metadata, err := svcCtx.LighterCache.GetOrderBookMetadata(ctx, symbol)
		if err != nil {
			return decimal.Zero, err
		}

		return svcCtx.LighterClient.GetLastTradePrice(ctx, uint(metadata.MarketID))
	default:
		return decimal.Zero, errors.New("exchange unsupported")
	}
}
