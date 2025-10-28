package handler

import (
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
)

func InitRoutes(svcCtx *svc.ServiceContext, router *pathrouter.Router) {
	NewCreateStrategyHandler(svcCtx).AddRouter(router)
	NewDeleteStrategyHandler(svcCtx).AddRouter(router)
	NewStrategyDetailsHandler(svcCtx).AddRouter(router)
	NewStrategyListHandler(svcCtx).AddRouter(router)
	NewStrategySettingsHandler(svcCtx).AddRouter(router)
	NewStrategySwitchHandler(svcCtx).AddRouter(router)
}
