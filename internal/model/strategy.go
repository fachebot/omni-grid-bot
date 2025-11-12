package model

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/predicate"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/shopspring/decimal"
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
		SetOwner(args.Owner).
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
		SetNillableSlippageBps(args.SlippageBps).
		SetEnableAutoExit(args.EnableAutoExit).
		SetEnablePushNotification(args.EnablePushNotification).
		SetNillableEnablePushMatchedNotification(args.EnablePushMatchedNotification).
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

func (m *StrategyModel) FindAllByOwner(ctx context.Context, owner int64, offset, limit int) ([]*ent.Strategy, int, error) {
	q := m.client.Query().Where(strategy.OwnerEQ(owner))
	count, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	data, err := q.Order(strategy.ByID(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return data, count, nil
}

func (m *StrategyModel) FindAllByActiveStatus(ctx context.Context, offset, limit int) ([]*ent.Strategy, error) {
	return m.client.Query().
		Where(strategy.StatusEQ(strategy.StatusActive)).
		Order(strategy.ByID()).
		Offset(offset).
		Limit(limit).
		All(ctx)
}

func (m *StrategyModel) FindAllByExchangeAndExchangeAPIKeyAndSymbol(ctx context.Context, exchange, exchangeAPIKey, symbol string) ([]*ent.Strategy, error) {
	ps := []predicate.Strategy{
		strategy.ExchangeEQ(exchange),
		strategy.ExchangeApiKeyEQ(exchangeAPIKey),
		strategy.SymbolEQ(symbol),
	}
	return m.client.Query().Where(ps...).All(ctx)
}

func (m *StrategyModel) UpdateStatus(ctx context.Context, id int, newValue strategy.Status) error {
	return m.client.UpdateOneID(id).SetStatus(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateExchange(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetExchange(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateAccount(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetAccount(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateExchangeAPIKey(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetExchangeApiKey(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateExchangeSecretKey(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetExchangeSecretKey(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateExchangePassphrase(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetExchangePassphrase(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateGridMode(ctx context.Context, id int, newValue strategy.Mode) error {
	return m.client.UpdateOneID(id).SetMode(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateMarginMode(ctx context.Context, id int, newValue strategy.MarginMode) error {
	return m.client.UpdateOneID(id).SetMarginMode(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateQuantityMode(ctx context.Context, id int, newValue strategy.QuantityMode) error {
	return m.client.UpdateOneID(id).SetQuantityMode(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateLeverage(ctx context.Context, id int, newValue int) error {
	return m.client.UpdateOneID(id).SetLeverage(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateGridNum(ctx context.Context, id int, newValue int) error {
	return m.client.UpdateOneID(id).SetGridNum(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateSymbol(ctx context.Context, id int, newValue string) error {
	return m.client.UpdateOneID(id).SetSymbol(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateSlippageBps(ctx context.Context, id int, newValue int) error {
	return m.client.UpdateOneID(id).SetSlippageBps(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateInitialOrderSize(ctx context.Context, id int, newValue decimal.Decimal) error {
	return m.client.UpdateOneID(id).SetInitialOrderSize(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdatePriceLower(ctx context.Context, id int, newValue decimal.Decimal) error {
	return m.client.UpdateOneID(id).SetPriceLower(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdatePriceUpper(ctx context.Context, id int, newValue decimal.Decimal) error {
	return m.client.UpdateOneID(id).SetPriceUpper(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateEnablePushNotification(ctx context.Context, id int, newValue bool) error {
	return m.client.UpdateOneID(id).SetEnablePushNotification(newValue).Exec(ctx)
}

func (m *StrategyModel) UpdateEnablePushMatchedNotification(ctx context.Context, id int, newValue bool) error {
	return m.client.UpdateOneID(id).SetEnablePushMatchedNotification(newValue).Exec(ctx)
}

func (m *StrategyModel) Delete(ctx context.Context, id int) error {
	return m.client.DeleteOneID(id).Exec(ctx)
}
