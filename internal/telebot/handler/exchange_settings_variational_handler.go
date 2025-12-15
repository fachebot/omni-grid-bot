package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fachebot/omni-grid-bot/internal/cache"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/fachebot/omni-grid-bot/internal/util/ethutil"

	"github.com/samber/lo"
	tele "gopkg.in/telebot.v4"
)

type VariationalSettingsOption int

var (
	VariationalSettingsOptionDexAccount    VariationalSettingsOption = 1
	VariationalSettingsOptionDexPrivateKey VariationalSettingsOption = 2
)

type ExchangeSettingsVariationalHandler struct {
	svcCtx *svc.ServiceContext
}

func NewExchangeSettingsVariationalHandler(svcCtx *svc.ServiceContext) *ExchangeSettingsVariationalHandler {
	return &ExchangeSettingsVariationalHandler{svcCtx: svcCtx}
}

func (h ExchangeSettingsVariationalHandler) FormatPath(guid string, option *VariationalSettingsOption) string {
	if option == nil {
		return fmt.Sprintf("/var/%s/settings", guid)
	}
	return fmt.Sprintf("/var/%s/settings/%d", guid, *option)
}

func (h *ExchangeSettingsVariationalHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/var/{uuid}/settings", h.handle)
	router.HandleFunc("/var/{uuid}/settings/{option}", h.handle)
}

func (h *ExchangeSettingsVariationalHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[ExchangeSettingsVariationalHandler] 查询策略信息失败, id: %s, %v", guid, err)
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

	// 更新交易所
	defaultExchange := exchange.Variational
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
		return DisplayExchangeSettingsVariationalSettings(ctx, h.svcCtx, userId, update, record)
	}

	// 更新交易所设置
	optionValue, err := strconv.Atoi(option)
	if err != nil {
		return DisplayExchangeSettingsVariationalSettings(ctx, h.svcCtx, userId, update, record)
	}
	switch VariationalSettingsOption(optionValue) {
	case VariationalSettingsOptionDexAccount:
		return h.handleDexAccount(ctx, userId, update, record)
	case VariationalSettingsOptionDexPrivateKey:
		return h.handleDexPrivateKey(ctx, userId, update, record)
	}

	return nil
}

func DisplayExchangeSettingsVariationalSettings(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy) error {
	// 测试连通性
	connectStatus := "🔴"
	if testVariationalConnectivity(ctx, svcCtx, record) == nil {
		connectStatus = "🟢"
	}

	statusText := func(s string) string {
		return lo.If(s != "", "✅").Else("⬜")
	}

	name := util.StrategyName(record)
	text := fmt.Sprintf("*%s* | 交易所配置 `%s`", svcCtx.Config.AppName, name)
	text += "\n\n「调整设置, 优化您的跟单体验」"

	dexAccount := record.ExchangeApiKey
	dexPrivateKey := record.ExchangeSecretKey
	h := ExchangeSettingsVariationalHandler{}
	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: fmt.Sprintf("%s variational", connectStatus), Data: ExchangeSelectorHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: fmt.Sprintf("%s 账户地址", statusText(dexAccount)), Data: h.FormatPath(record.GUID, &VariationalSettingsOptionDexAccount)},
				{Text: fmt.Sprintf("%s 账户私钥", statusText(dexPrivateKey)), Data: h.FormatPath(record.GUID, &VariationalSettingsOptionDexPrivateKey)},
			},

			{
				{Text: "◀️ 返回上级", Data: StrategySettingsHandler{}.FormatPath(record.GUID)},
				{Text: "⏪ 返回主页", Data: "/home"},
			},
		},
	}

	_, err := util.ReplyMessage(svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[DisplayExchangeSettingsVariationalSettings] 生成Variational设置界面失败, %v", err)
	}
	return nil
}

func (h *ExchangeSettingsVariationalHandler) refreshSettingsMessage(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	chatId := update.Message.Chat.ID
	if update.Message.ReplyTo == nil {
		return DisplayExchangeSettingsVariationalSettings(ctx, h.svcCtx, userId, update, record)
	} else {
		route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyTo.ID)
		if ok && route.Context != nil {
			return DisplayExchangeSettingsVariationalSettings(ctx, h.svcCtx, userId, tele.Update{Message: route.Context}, record)
		}
		return DisplayExchangeSettingsVariationalSettings(ctx, h.svcCtx, userId, update, record)
	}
}

func (h *ExchangeSettingsVariationalHandler) handleDexAccount(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写Variational账户地址。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[ExchangeSettingsVariationalHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &VariationalSettingsOptionDexAccount), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		chatId := update.Message.Chat.ID
		dexAccount := common.HexToAddress(update.Message.Text).Hex()
		if !strings.EqualFold(dexAccount, dexAccount) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效Variational账户地址", 3)
			return nil
		}

		if record.Symbol != "" {
			result, err := h.svcCtx.StrategyModel.FindAllByExchangeAndAccountAndSymbol(ctx, exchange.Variational, dexAccount, record.Symbol)
			if err != nil || len(result) > 0 {
				text := "❌ 此Variational账户地址已被其他网格策略使用"
				chatId := util.ChatId(update.Callback.Message.Chat.ID)
				util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)
				return nil
			}
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err := util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
			m := model.NewStrategyModel(tx.Strategy)
			if err := m.UpdateAccount(ctx, record.ID, dexAccount); err != nil {
				return err
			}

			if err := m.UpdateExchangeSecretKey(ctx, record.ID, ""); err != nil {
				return err
			}

			return m.UpdateExchangeAPIKey(ctx, record.ID, dexAccount)
		})
		if err == nil {
			record.ExchangeApiKey = dexAccount
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[ExchangeSettingsVariationalHandler] 更新配置[ExchangeAPIKey]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		// 刷新用户界面
		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *ExchangeSettingsVariationalHandler) handleDexPrivateKey(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写Variational账户私钥。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[ExchangeSettingsVariationalHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, &VariationalSettingsOptionDexPrivateKey), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		chatId := update.Message.Chat.ID
		dexPrivateKey := update.Message.Text
		_, address, err := ethutil.ParsePrivateKey(dexPrivateKey)
		if err != nil {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效Variational账户私钥", 3)
			return nil
		}

		if record.Account != "" && !strings.EqualFold(record.Account, address.Hex()) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ Variational账户地址和私钥不匹配", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateExchangeSecretKey(ctx, record.ID, dexPrivateKey)
		if err == nil {
			record.ExchangeSecretKey = dexPrivateKey
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[ExchangeSettingsVariationalHandler] 更新配置[ExchangeSecretKey]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		// 刷新用户界面
		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}
