package model

import (
	"context"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/order"
)

type OrderModel struct {
	client *ent.OrderClient
}

func NewOrderModel(client *ent.OrderClient) *OrderModel {
	return &OrderModel{client: client}
}

func (m *OrderModel) Upsert(ctx context.Context, args ent.Order) error {
	existing, err := m.client.Query().
		Where(order.ExchangeEQ(args.Exchange), order.SymbolEQ(args.Symbol), order.OrderIdEQ(args.OrderId)).
		First(ctx)
	if ent.IsNotFound(err) {
		return m.client.Create().
			SetExchange(args.Exchange).
			SetAccount(args.Account).
			SetSymbol(args.Symbol).
			SetOrderId(args.OrderId).
			SetClientOrderId(args.ClientOrderId).
			SetSide(args.Side).
			SetPrice(args.Price).
			SetBaseAmount(args.BaseAmount).
			SetFilledBaseAmount(args.FilledBaseAmount).
			SetFilledQuoteAmount(args.FilledQuoteAmount).
			SetStatus(args.Status).
			SetTimestamp(args.Timestamp).
			Exec(ctx)
	}

	if err != nil {
		return err
	}

	if args.Timestamp > existing.Timestamp ||
		(existing.Status == order.StatusOpen && existing.Status != args.Status) {
		return m.client.Update().
			SetSide(args.Side).
			SetPrice(args.Price).
			SetBaseAmount(args.BaseAmount).
			SetFilledBaseAmount(args.FilledBaseAmount).
			SetFilledQuoteAmount(args.FilledQuoteAmount).
			SetStatus(args.Status).
			SetTimestamp(args.Timestamp).
			Where(order.ExchangeEQ(args.Exchange), order.SymbolEQ(args.Symbol), order.OrderIdEQ(args.OrderId)).
			Exec(ctx)
	}

	return nil
}

func (m *OrderModel) FindOneByAccountClientOrderId(ctx context.Context, exchange string, account string, clientOrderId int64) (*ent.Order, error) {
	return m.client.Query().
		Where(order.ExchangeEQ(exchange), order.AccountEQ(account), order.ClientOrderIdEQ(clientOrderId)).
		First(ctx)
}

func (m *OrderModel) FindAllByAccountClientOrderIds(ctx context.Context, exchange string, account string, clientOrderIds []int64) ([]*ent.Order, error) {
	if len(clientOrderIds) == 0 {
		return nil, nil
	}

	return m.client.Query().
		Where(order.ExchangeEQ(exchange), order.AccountEQ(account), order.ClientOrderIdIn(clientOrderIds...)).
		All(ctx)
}
