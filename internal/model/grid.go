package model

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/grid"
)

type GridModel struct {
	client *ent.GridClient
}

func NewGridModel(client *ent.GridClient) *GridModel {
	return &GridModel{client: client}
}

func (m *GridModel) CreateBulk(ctx context.Context, items []ent.Grid) error {
	builders := make([]*ent.GridCreate, 0, len(items))
	for _, item := range items {
		builder := m.client.Create().
			SetStrategyId(item.StrategyId).
			SetExchange(item.Exchange).
			SetSymbol(item.Symbol).
			SetAccount(item.Account).
			SetLevel(item.Level).
			SetPrice(item.Price).
			SetQuantity(item.Quantity).
			SetNillableBuyClientOrderId(item.BuyClientOrderId).
			SetNillableSellClientOrderId(item.SellClientOrderId)
		builders = append(builders, builder)
	}

	return m.client.CreateBulk(builders...).Exec(ctx)
}

func (m *GridModel) FindAllByStrategyIdOrderAsc(ctx context.Context, strategyId string) ([]*ent.Grid, error) {
	return m.client.Query().Where(grid.StrategyIdEQ(strategyId)).Order(grid.ByLevel(sql.OrderAsc())).All(ctx)
}

func (m *GridModel) UpdateBuyClientOrderId(ctx context.Context, id int, newValue *int64) error {
	if newValue == nil {
		return m.client.UpdateOneID(id).ClearBuyClientOrderId().Exec(ctx)
	}
	return m.client.UpdateOneID(id).SetBuyClientOrderId(*newValue).Exec(ctx)
}

func (m *GridModel) UpdateSellClientOrderId(ctx context.Context, id int, newValue *int64) error {
	if newValue == nil {
		return m.client.UpdateOneID(id).ClearSellClientOrderId().Exec(ctx)
	}
	return m.client.UpdateOneID(id).SetSellClientOrderId(*newValue).Exec(ctx)
}

func (m *GridModel) DeleteByStrategyId(ctx context.Context, strategyId string) error {
	_, err := m.client.Delete().Where(grid.StrategyIdEQ(strategyId)).Exec(ctx)
	return err
}
