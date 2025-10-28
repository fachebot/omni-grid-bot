package handler

import (
	"context"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type CreateStrategyHandler struct {
	svcCtx *svc.ServiceContext
}

func NewCreateStrategyHandler(svcCtx *svc.ServiceContext) *CreateStrategyHandler {
	return &CreateStrategyHandler{svcCtx: svcCtx}
}

func (h CreateStrategyHandler) FormatPath() string {
	return "/strategy/create"
}

func (h *CreateStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/create", h.handle)
}

func (h *CreateStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
