package helper

import (
	"context"
	"errors"
	"strconv"

	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/samber/lo"
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

func GetAccountInfo(ctx context.Context, svcCtx *svc.ServiceContext, exchangeType, account string) (*exchange.Account, error) {
	switch exchangeType {
	case exchange.Lighter:
		return GetLighterAccountInfo(ctx, svcCtx, account)
	default:
		return nil, errors.New("exchange unsupported")
	}
}

func GetLighterAccountInfo(ctx context.Context, svcCtx *svc.ServiceContext, account string) (*exchange.Account, error) {
	accountIndex, err := strconv.ParseInt(account, 10, 64)
	if err != nil {
		return nil, err
	}

	accounts, err := svcCtx.LighterClient.GetAccountByIndex(ctx, accountIndex)
	if err != nil {
		return nil, err
	}

	accountInfo, ok := lo.Find(accounts.Accounts, func(item *lighter.Account) bool {
		return item.Index == accountIndex
	})
	if !ok {
		return nil, errors.New("account not found")
	}

	ret := exchange.Account{
		AvailableBalance: accountInfo.AvailableBalance,
		Collateral:       accountInfo.Collateral,
		Positions:        make([]*exchange.Position, 0),
		TotalAssetValue:  accountInfo.TotalAssetValue,
		CrossAssetValue:  accountInfo.CrossAssetValue,
	}
	for _, item := range accountInfo.Positions {
		if item.Position.LessThanOrEqual(decimal.Zero) {
			continue
		}

		ret.Positions = append(ret.Positions, &exchange.Position{
			Symbol:                item.Symbol,
			InitialMarginFraction: item.InitialMarginFraction,
			Side:                  exchange.PositionSide(item.Sign),
			Position:              item.Position,
			AvgEntryPrice:         item.AvgEntryPrice,
			PositionValue:         item.PositionValue,
			UnrealizedPnl:         item.UnrealizedPnl,
			RealizedPnl:           item.RealizedPnl,
			LiquidationPrice:      item.LiquidationPrice,
			TotalFundingPaidOut:   item.TotalFundingPaidOut,
			MarginMode:            exchange.MarginMode(item.MarginMode),
			AllocatedMargin:       item.AllocatedMargin,
		})
	}
	return &ret, nil
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
