package helper

import (
	"context"
	"errors"
	"strconv"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/lighter"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

type MarketMetadata struct {
	MinBaseAmount          decimal.Decimal // 最小基础货币数量
	MinQuoteAmount         decimal.Decimal // 最小计价货币数量
	SupportedSizeDecimals  uint8           // 支持的数量小数位数
	SupportedPriceDecimals uint8           // 支持的价格小数位数
	SupportedQuoteDecimals uint8           // 支持的计价小数位数
}

func GetAccountInfo(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) (*exchange.Account, error) {
	switch record.Exchange {
	case exchange.Lighter:
		return GetLighterAccountInfo(ctx, svcCtx, record.Account)
	case exchange.Paradex:
		return GetParadexAccountInfo(ctx, svcCtx, record)
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
		Positions:        make([]*exchange.Position, 0),
		TotalAssetValue:  accountInfo.TotalAssetValue,
	}
	for _, item := range accountInfo.Positions {
		if item.Position.LessThanOrEqual(decimal.Zero) {
			continue
		}

		ret.Positions = append(ret.Positions, &exchange.Position{
			Symbol:              item.Symbol,
			Side:                exchange.PositionSide(item.Sign),
			Position:            item.Position,
			AvgEntryPrice:       item.AvgEntryPrice,
			UnrealizedPnl:       item.UnrealizedPnl,
			RealizedPnl:         item.RealizedPnl,
			LiquidationPrice:    item.LiquidationPrice,
			TotalFundingPaidOut: item.TotalFundingPaidOut,
			MarginMode:          exchange.MarginMode(item.MarginMode),
		})
	}
	return &ret, nil
}

func GetParadexAccountInfo(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) (*exchange.Account, error) {
	client, err := GetParadexClient(svcCtx, record)
	if err != nil {
		return nil, err
	}

	positions, err := client.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	marginConfigs, err := client.GetMarginConfig(ctx)
	if err != nil {
		return nil, err
	}

	accountSummaries, err := client.GetAccountSummaries(ctx)
	if err != nil {
		return nil, err
	}

	marginModeMap := make(map[string]paradex.MarginType)
	for _, item := range marginConfigs.Configs {
		marginModeMap[item.Market] = item.MarginType
	}

	account := exchange.Account{
		AvailableBalance: accountSummaries[0].FreeCollateral,
		Positions:        make([]*exchange.Position, 0, len(positions.Results)),
	}
	for _, item := range accountSummaries {
		account.TotalAssetValue = account.TotalAssetValue.Add(item.AccountValue)
	}

	for _, item := range positions.Results {
		if item.Size.IsZero() {
			continue
		}

		symbol, err := paradex.ParseUsdPerpMarket(item.Market)
		if err != nil {
			continue
		}

		liquidationPrice := decimal.Zero
		if item.LiquidationPrice != "" {
			liquidationPrice, _ = decimal.NewFromString(item.LiquidationPrice)
		}

		marginMode := exchange.MarginModeCross
		if v, ok := marginModeMap[item.Market]; ok {
			marginMode = lo.If(v == paradex.MarginTypeCross, exchange.MarginModeCross).Else(exchange.MarginModeIsolated)
		}

		account.Positions = append(account.Positions, &exchange.Position{
			Symbol:              symbol,
			Side:                lo.If(item.Side == paradex.PositionSideLong, exchange.PositionSideLong).Else(exchange.PositionSideShort),
			Position:            item.Size.Abs(),
			AvgEntryPrice:       item.AverageEntryPrice,
			UnrealizedPnl:       item.UnrealizedFundingPnl,
			RealizedPnl:         item.RealizedPositionalPnl,
			LiquidationPrice:    liquidationPrice,
			TotalFundingPaidOut: item.RealizedPositionalFundingPnl.Add(item.UnrealizedFundingPnl),
			MarginMode:          marginMode,
		})
	}

	return &account, nil
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
	case exchange.Paradex:
		marketSummary, err := svcCtx.ParadexClient.GetMarketSummary(ctx, paradex.FormatUsdPerpMarket(symbol))
		if err != nil {
			return decimal.Zero, err
		}

		if len(marketSummary.Results) == 0 {
			return decimal.Zero, nil
		}

		return marketSummary.Results[0].LastTradedPrice, nil
	default:
		return decimal.Zero, errors.New("exchange unsupported")
	}
}
