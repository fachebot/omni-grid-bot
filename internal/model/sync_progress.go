package model

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/syncprogress"
)

type SyncProgressModel struct {
	client *ent.SyncProgressClient
}

func NewSyncProgressModel(client *ent.SyncProgressClient) *SyncProgressModel {
	return &SyncProgressModel{client: client}
}

func (m *SyncProgressModel) Ensure(ctx context.Context, exchange, account string) (*ent.SyncProgress, error) {
	r, err := m.client.Query().
		Where(syncprogress.ExchangeEQ(exchange), syncprogress.AccountEQ(account)).
		First(ctx)
	if err == nil {
		return r, nil
	}

	if !ent.IsNotFound(err) {
		return nil, err
	}

	return m.client.Create().SetExchange(exchange).SetAccount(account).SetTimestamp(0).Save(ctx)
}

func (m *SyncProgressModel) UpdateTimestampByAccount(ctx context.Context, exchange, account string, timestamp int64) error {
	return m.client.Update().Where(syncprogress.ExchangeEQ(exchange), syncprogress.AccountEQ(account)).SetTimestamp(timestamp).Exec(ctx)
}
