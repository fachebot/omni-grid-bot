package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	tele "gopkg.in/telebot.v4"
)

type StopType string

var (
	StopTypeStop  StopType = "stop"
	StopTypeClose StopType = "close"
)

type StrategySwitchHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategySwitchHandler(svcCtx *svc.ServiceContext) *StrategySwitchHandler {
	return &StrategySwitchHandler{svcCtx: svcCtx}
}

func (h StrategySwitchHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/switch/%s", guid)
}

func (h *StrategySwitchHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/switch/{uuid}", h.handle)
	router.HandleFunc("/strategy/switch/{uuid}/{stop}", h.handle)
}

func (h *StrategySwitchHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	return nil
}
