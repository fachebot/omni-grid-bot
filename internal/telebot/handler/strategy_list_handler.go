package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type StrategyListHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategyListHandler(svcCtx *svc.ServiceContext) *StrategyListHandler {
	return &StrategyListHandler{svcCtx: svcCtx}
}

func (h StrategyListHandler) FormatPath(page int) string {
	return fmt.Sprintf("/strategy/list/%d", page)
}

func (h *StrategyListHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/list", h.handle)
	router.HandleFunc("/strategy/list/{page:[0-9]+}", h.handle)
}

func (h *StrategyListHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
