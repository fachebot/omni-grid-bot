package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

type ExchangeSettingsHandler struct {
	svcCtx *svc.ServiceContext
}

func NewExchangeSettingsHandler(svcCtx *svc.ServiceContext) *ExchangeSettingsHandler {
	return &ExchangeSettingsHandler{svcCtx: svcCtx}
}

func (h ExchangeSettingsHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/exchange/%s/settings", guid)
}

func (h *ExchangeSettingsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/exchange/{uuid}/settings", h.handle)
}

func (h *ExchangeSettingsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[ExchangeSettingsHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	// 设置默认交易所
	if record.Exchange == "" {
		err = h.svcCtx.StrategyModel.UpdateExchange(ctx, record.ID, exchange.Lighter)
		if err != nil {
			logger.Errorf("[ExchangeSettingsHandler] 更新配置[Exchange]失败, %v", err)

			text := "❌ 服务器内部错误, 请稍后重试"
			chatId := util.ChatId(update.Callback.Message.Chat.ID)
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)
			return nil
		}

		record.Exchange = exchange.Lighter
	}

	// 返回交易所配置
	switch record.Exchange {
	case exchange.Lighter:
		return NewExchangeSettingsLighterHandler(h.svcCtx).handle(ctx, vars, userId, update)
	}

	return nil
}
