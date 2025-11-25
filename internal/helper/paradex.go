package helper

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/svc"
)

type ParadexOrderHelper struct {
	svcCtx     *svc.ServiceContext
	userClient *paradex.UserClient
}

func GetParadexClient(svcCtx *svc.ServiceContext, record *ent.Strategy) (*paradex.UserClient, error) {
	dexAccount := record.ExchangeApiKey
	dexPrivateKey := record.ExchangeSecretKey
	return paradex.NewUserClient(svcCtx.ParadexClient, dexAccount, dexPrivateKey), nil
}

func NewParadexOrderHelper(svcCtx *svc.ServiceContext, userClient *paradex.UserClient) *ParadexOrderHelper {
	return &ParadexOrderHelper{svcCtx: svcCtx, userClient: userClient}
}

func (h *ParadexOrderHelper) SyncInactiveOrders(ctx context.Context) error {
	return nil
}
