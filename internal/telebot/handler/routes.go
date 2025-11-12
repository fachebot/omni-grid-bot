package handler

import (
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
)

func InitRoutes(svcCtx *svc.ServiceContext, router *pathrouter.Router) {
	NewCompletedTradesHandler(svcCtx).AddRouter(router)
	NewCreateStrategyHandler(svcCtx).AddRouter(router)
	NewDeleteStrategyHandler(svcCtx).AddRouter(router)
	NewStrategyDetailsHandler(svcCtx).AddRouter(router)
	NewStrategyListHandler(svcCtx).AddRouter(router)
	NewStrategySettingsHandler(svcCtx).AddRouter(router)
	NewStrategySwitchHandler(svcCtx).AddRouter(router)
	NewExchangeSelectorHandler(svcCtx).AddRouter(router)
	NewExchangeSettingsHandler(svcCtx).AddRouter(router)
	NewExchangeSettingsLighterHandler(svcCtx).AddRouter(router)
}
