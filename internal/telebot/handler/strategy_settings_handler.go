package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type SettingsOption int

var (
	SettingsOptionMaxLeverage      SettingsOption = 1
	SettingsOptionExchangeSettings SettingsOption = 2
)

type StrategySettingsHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategySettingsHandler(svcCtx *svc.ServiceContext) *StrategySettingsHandler {
	return &StrategySettingsHandler{svcCtx: svcCtx}
}

func (h StrategySettingsHandler) FormatPath(guid string, option ...SettingsOption) string {
	if len(option) == 0 {
		return fmt.Sprintf("/strategy/settings/%s", guid)
	}
	return fmt.Sprintf("/strategy/settings/%s/%d", guid, option[0])
}

func (h *StrategySettingsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/settings/{uuid}", h.handle)
	router.HandleFunc("/strategy/settings/{uuid}/{option}", h.handle)
}

func (h *StrategySettingsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
