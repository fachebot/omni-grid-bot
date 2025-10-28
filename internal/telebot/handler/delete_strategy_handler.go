package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type DeleteStrategyHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeleteStrategyHandler(svcCtx *svc.ServiceContext) *DeleteStrategyHandler {
	return &DeleteStrategyHandler{svcCtx: svcCtx}
}

func (h DeleteStrategyHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/delete/%s", guid)
}

func (h *DeleteStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/delete/{uuid}", h.handle)
	router.HandleFunc("/strategy/delete/{uuid}/{confirm}", h.handle)
}

func (h *DeleteStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
