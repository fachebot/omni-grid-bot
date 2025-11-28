package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

type ExchangeSelectorHandler struct {
	svcCtx *svc.ServiceContext
}

func NewExchangeSelectorHandler(svcCtx *svc.ServiceContext) *ExchangeSelectorHandler {
	return &ExchangeSelectorHandler{svcCtx: svcCtx}
}

func (h ExchangeSelectorHandler) FormatPath(guid string, ex ...string) string {
	if len(ex) == 0 {
		return fmt.Sprintf("/exchange/%s/select", guid)
	}
	return fmt.Sprintf("/exchange/%s/select/%s", guid, ex[0])
}

func (h *ExchangeSelectorHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/exchange/{uuid}/select", h.handle)
	router.HandleFunc("/exchange/{uuid}/select/{exchange}", h.handle)
}

func (h *ExchangeSelectorHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	// 查询策略信息
	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[ExchangeSelectorHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return nil
	}

	if record.Status != strategy.StatusInactive {
		chat, ok := util.GetChat(update)
		if ok {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, "❌ 策略运行中不允许修改此参数", 3)
		}
		return nil
	}

	name := StrategyName(record)
	text := fmt.Sprintf("*%s* | 交易平台 `%s`", h.svcCtx.Config.AppName, name)
	text += "\n\n请选择运行网格策略的交易平台:"

	// 返回交易所列表
	value, ok := vars["exchange"]
	if !ok {
		replyMarkup := &tele.ReplyMarkup{
			InlineKeyboard: [][]tele.InlineButton{
				{
					{Text: exchange.Lighter, Data: h.FormatPath(guid, exchange.Lighter)},
				},
				{
					{Text: exchange.Paradex, Data: h.FormatPath(guid, exchange.Paradex)},
				},
			},
		}

		_, err = util.ReplyMessage(h.svcCtx.Bot, update, text, replyMarkup)
		if err != nil {
			logger.Debugf("[ExchangeSelectorHandler] 生成交易所列表UI失败, %v", err)
		}
		return nil
	}

	if !exchange.IsValidExchanges(value) {
		return nil
	}

	// 更新交易平台
	if record.Exchange == "" || record.Exchange != value {
		err = util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
			m := model.NewStrategyModel(tx.Strategy)
			err = m.UpdateExchange(ctx, record.ID, value)
			if err != nil {
				return err
			}

			if err = m.UpdateExchangeAPIKey(ctx, record.ID, ""); err != nil {
				return err
			}

			if err = m.UpdateExchangeSecretKey(ctx, record.ID, ""); err != nil {
				return err
			}

			if err = m.UpdateExchangePassphrase(ctx, record.ID, ""); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			logger.Errorf("[ExchangeSelectorHandler] 更新配置[Exchange]失败, %v", err)

			text := "❌ 服务器内部错误, 请稍后重试"
			chatId := util.ChatId(update.Callback.Message.Chat.ID)
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)
			return nil
		}
	}

	// 返回交易所账户配置
	switch value {
	case exchange.Lighter:
		return NewExchangeSettingsLighterHandler(h.svcCtx).handle(ctx, vars, userId, update)
	case exchange.Paradex:
		return NewExchangeSettingsParadexHandler(h.svcCtx).handle(ctx, vars, userId, update)
	}

	return nil
}
