package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type CompletedTradesHandler struct {
	svcCtx *svc.ServiceContext
}

func NewCompletedTradesHandler(svcCtx *svc.ServiceContext) *CompletedTradesHandler {
	return &CompletedTradesHandler{svcCtx: svcCtx}
}

func (h CompletedTradesHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/trades/%s", guid)
}

func (h *CompletedTradesHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/trades/{uuid}", h.handle)
}

func (h *CompletedTradesHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
