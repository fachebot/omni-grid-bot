package model

import (
	"context"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/strategy"
)

type StrategyModel struct {
	client *ent.StrategyClient
}

func NewStrategyModel(client *ent.StrategyClient) *StrategyModel {
	return &StrategyModel{client: client}
}

func (m *StrategyModel) Save(ctx context.Context, args ent.Strategy) (*ent.Strategy, error) {
	return m.client.Create().
		SetGUID(args.GUID).
		SetExchange(args.Exchange).
		SetSymbol(args.Symbol).
		SetAccount(args.Account).
		SetMode(args.Mode).
		SetMarginMode(args.MarginMode).
		SetQuantityMode(args.QuantityMode).
		SetPriceUpper(args.PriceUpper).
		SetPriceLower(args.PriceLower).
		SetGridNum(args.GridNum).
		SetLeverage(args.Leverage).
		SetInitialOrderSize(args.InitialOrderSize).
		SetStopLossRatio(args.StopLossRatio).
		SetTakeProfitRatio(args.TakeProfitRatio).
		SetEnableAutoBuy(args.EnableAutoBuy).
		SetEnableAutoSell(args.EnableAutoSell).
		SetEnableAutoExit(args.EnableAutoExit).
		SetEnablePushNotification(args.EnablePushNotification).
		SetNillableLastLowerThresholdAlertTime(args.LastLowerThresholdAlertTime).
		SetNillableLastUpperThresholdAlertTime(args.LastUpperThresholdAlertTime).
		SetStatus(args.Status).
		SetExchangeApiKey(args.ExchangeApiKey).
		SetExchangeSecretKey(args.ExchangeSecretKey).
		SetExchangePassphrase(args.ExchangePassphrase).
		Save(ctx)
}

func (m *StrategyModel) FindOneByGUID(ctx context.Context, guid string) (*ent.Strategy, error) {
	return m.client.Query().Where(strategy.GUIDEQ(guid)).First(ctx)
}

func (m *StrategyModel) UpdateStatus(ctx context.Context, id int, newValue strategy.Status) error {
	return m.client.UpdateOneID(id).SetStatus(newValue).Exec(ctx)
}
