package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fachebot/omni-grid-bot/internal/cache"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"

	"github.com/samber/lo"
	tele "gopkg.in/telebot.v4"
)

type LighterSettingsOption int

var (
	LighterSettingsOptionAccountIndex     LighterSettingsOption = 1
	LighterSettingsOptionApiKeyPrivateKey LighterSettingsOption = 2
	LighterSettingsOptionApiKeyIndex      LighterSettingsOption = 3
)

type ExchangeSettingsLighterHandler struct {
	svcCtx *svc.ServiceContext
}

func NewExchangeSettingsLighterHandler(svcCtx *svc.ServiceContext) *ExchangeSettingsLighterHandler {
	return &ExchangeSettingsLighterHandler{svcCtx: svcCtx}
}

func (h ExchangeSettingsLighterHandler) FormatPath(guid string, option *LighterSettingsOption) string {
	if option == nil {
		return fmt.Sprintf("/lighter/%s/settings", guid)
	}
	return fmt.Sprintf("/lighter/%s/settings/%d", guid, *option)
}

func (h *ExchangeSettingsLighterHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/lighter/{uuid}/settings", h.handle)
	router.HandleFunc("/lighter/{uuid}/settings/{option}", h.handle)
}

func (h *ExchangeSettingsLighterHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[ExchangeSettingsLighterHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return nil
	}

	// 更新交易所
	defaultExchange := exchange.Lighter
	err = h.svcCtx.StrategyModel.UpdateExchange(ctx, record.ID, defaultExchange)
	if err != nil {
		logger.Errorf("[ExchangeSettingsHandler] 更新配置[Exchange]失败, %v", err)

		text := "❌ 服务器内部错误, 请稍后重试"
		chatId := util.ChatId(update.Callback.Message.Chat.ID)
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)
		return nil
	}

	record.Exchange = defaultExchange

	// 显示设置界面
	option, ok := vars["option"]
	if !ok {
		return DisplayExchangeSettingsLighterSettings(ctx, h.svcCtx, userId, update, record)
	}

	// 更新交易所设置
	optionValue, err := strconv.Atoi(option)
	if err != nil {
		return DisplayExchangeSettingsLighterSettings(ctx, h.svcCtx, userId, update, record)
	}
	switch LighterSettingsOption(optionValue) {
	case LighterSettingsOptionAccountIndex:
		return h.handleAccountIndex(ctx, userId, update, record)
	case LighterSettingsOptionApiKeyPrivateKey:
		return h.handleApiKeyPrivateKey(ctx, userId, update, record)
	case LighterSettingsOptionApiKeyIndex:
		return h.handleApiKeyIndex(ctx, userId, update, record)
	}

	return nil
}

func DisplayExchangeSettingsLighterSettings(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy) error {
	// 测试连通性
	accountIndexText := ""
	_, err := strconv.Atoi(record.ExchangeApiKey)
	if err == nil {
		accountIndexText = record.ExchangeApiKey
	}

	apiKeyIndexText := ""
	_, err = strconv.Atoi(record.ExchangeSecretKey)
	if err == nil {
		apiKeyIndexText = record.ExchangeSecretKey
	}

	connectStatus := "🔴"
	if testLighterConnectivity(ctx, svcCtx, record) == nil {
		connectStatus = "🟢"
	}

	statusText := func(s string) string {
		return lo.If(s != "", "✅").Else("⬜")
	}

	name := StrategyName(record)
	text := fmt.Sprintf("*%s* | 交易所配置 `%s`", svcCtx.Config.AppName, name)
	text += "\n\n「调整设置, 优化您的跟单体验」"

	h := ExchangeSettingsLighterHandler{}
	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: fmt.Sprintf("%s lighter", connectStatus), Data: ExchangeSelectorHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: fmt.Sprintf("%s AccountIndex", statusText(accountIndexText)), Data: h.FormatPath(record.GUID, &LighterSettingsOptionAccountIndex)},
				{Text: fmt.Sprintf("%s ApiKeyIndex", statusText(apiKeyIndexText)), Data: h.FormatPath(record.GUID, &LighterSettingsOptionApiKeyIndex)},
			},
			{
				{Text: fmt.Sprintf("%s ApiKeyPrivateKey", statusText(record.ExchangePassphrase)), Data: h.FormatPath(record.GUID, &LighterSettingsOptionApiKeyPrivateKey)},
			},
			{
				{Text: "◀️ 返回上级", Data: StrategySettingsHandler{}.FormatPath(record.GUID)},
				{Text: "⏪ 返回主页", Data: "/home"},
			},
		},
	}

	_, err = util.ReplyMessage(svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[DisplayExchangeSettingsLighterSettings] 生成Lighter设置界面失败, %v", err)
	}
	return nil
}

func (h *ExchangeSettingsLighterHandler) refreshSettingsMessage(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	chatId := update.Message.Chat.ID
	if update.Message.ReplyTo == nil {
		return DisplayExchangeSettingsLighterSettings(ctx, h.svcCtx, userId, update, record)
	} else {
		route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyTo.ID)
		if ok && route.Context != nil {
			return DisplayExchangeSettingsLighterSettings(ctx, h.svcCtx, userId, tele.Update{Message: route.Context}, record)
		}
		return DisplayExchangeSettingsLighterSettings(ctx, h.svcCtx, userId, update, record)
	}
}

func (h *ExchangeSettingsLighterHandler) handleAccountIndex(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写Lighter AccountIndex，值为大于0的整数。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[ExchangeSettingsLighterHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &LighterSettingsOptionAccountIndex), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		chatId := update.Message.Chat.ID
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d <= 0 {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效Lighter AccountIndex，值为大于0的整数", 3)
			return nil
		}

		apiKey := update.Message.Text
		if record.Symbol != "" {
			result, err := h.svcCtx.StrategyModel.FindAllByExchangeAndAccountAndSymbol(ctx, exchange.Lighter, apiKey, record.Symbol)
			if err != nil || len(result) > 0 {
				text := "❌ 此Lighter AccountIndex已被其他网格策略使用"
				chatId := util.ChatId(update.Callback.Message.Chat.ID)
				util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)
				return nil
			}
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
			m := model.NewStrategyModel(tx.Strategy)
			if err = m.UpdateAccount(ctx, record.ID, apiKey); err != nil {
				return err
			}
			return m.UpdateExchangeAPIKey(ctx, record.ID, apiKey)
		})
		if err == nil {
			record.ExchangeApiKey = apiKey
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[ExchangeSettingsLighterHandler] 更新配置[ExchangeAPIKey]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		// 刷新用户界面
		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *ExchangeSettingsLighterHandler) handleApiKeyIndex(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写Lighter ApiKeyIndex，值为大于等于0的整数。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[ExchangeSettingsLighterHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &LighterSettingsOptionApiKeyIndex), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		chatId := update.Message.Chat.ID
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d < 0 {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效Lighter ApiKeyIndex，值为大于等于0的整数", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateExchangeSecretKey(ctx, record.ID, update.Message.Text)
		if err == nil {
			record.ExchangeSecretKey = update.Message.Text
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[ExchangeSettingsLighterHandler] 更新配置[ExchangeSecretKey]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		// 刷新用户界面
		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *ExchangeSettingsLighterHandler) handleApiKeyPrivateKey(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写Lighter ApiKeyPrivateKey，值为长度80的字符串。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[ExchangeSettingsLighterHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &LighterSettingsOptionApiKeyPrivateKey), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		chatId := update.Message.Chat.ID
		if len(update.Message.Text) != 80 {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效Lighter ApiKeyPrivateKey，值为长度80的字符串", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err := h.svcCtx.StrategyModel.UpdateExchangePassphrase(ctx, record.ID, update.Message.Text)
		if err == nil {
			record.ExchangePassphrase = update.Message.Text
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[ExchangeSettingsLighterHandler] 更新配置[ExchangePassphrase]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		// 刷新用户界面
		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}
