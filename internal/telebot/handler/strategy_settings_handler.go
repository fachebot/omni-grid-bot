package handler

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/fachebot/omni-grid-bot/internal/cache"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	gridstrategy "github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/fachebot/omni-grid-bot/internal/util/format"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	tele "gopkg.in/telebot.v4"
)

type SettingsOption int

var (
	SettingsOptionLeverage                      SettingsOption = 1
	SettingsOptionGridNum                       SettingsOption = 2
	SettingsOptionGridMode                      SettingsOption = 3
	SettingsOptionMarginMode                    SettingsOption = 4
	SettingsOptionQuantityMode                  SettingsOption = 5
	SettingsOptionOrderSize                     SettingsOption = 6
	SettingsOptionPriceLower                    SettingsOption = 7
	SettingsOptionPriceUpper                    SettingsOption = 8
	SettingsOptionExchangeSettings              SettingsOption = 9
	SettingsOptionMarketSymbol                  SettingsOption = 10
	SettingsOptionSlippage                      SettingsOption = 11
	SettingsOptionEnablePushNotification        SettingsOption = 12
	SettingsOptionEnablePushMatchedNotification SettingsOption = 13
)

const (
	MaxShowGridNum = 10
)

type StrategySettingsHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategySettingsHandler(svcCtx *svc.ServiceContext) *StrategySettingsHandler {
	return &StrategySettingsHandler{svcCtx: svcCtx}
}

func (h StrategySettingsHandler) FormatPath(guid string, option ...SettingsOption) string {
	if len(option) == 0 {
		return fmt.Sprintf("/strategy/settings/%s", guid)
	}
	return fmt.Sprintf("/strategy/settings/%s/%d", guid, option[0])
}

func (h *StrategySettingsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/settings/{uuid}", h.handle)
	router.HandleFunc("/strategy/settings/{uuid}/{option}", h.handle)
}

func (h *StrategySettingsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[StrategySettingsHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return nil
	}

	option, ok := vars["option"]
	if !ok {
		return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
	}

	optionValue, err := strconv.Atoi(option)
	if err != nil {
		return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
	}

	if update.Callback != nil && record.Status != strategy.StatusInactive {
		allowList := []SettingsOption{
			SettingsOptionSlippage,
			SettingsOptionEnablePushNotification,
			SettingsOptionEnablePushMatchedNotification,
		}
		if lo.IndexOf(allowList, SettingsOption(optionValue)) == -1 {
			chatId := util.ChatId(update.Callback.Message.Chat.ID)
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, "❌ 策略运行中不允许修改此参数", 3)
			return nil
		}
	}

	switch SettingsOption(optionValue) {
	case SettingsOptionGridMode:
		return h.handleGridMode(ctx, userId, update, record)
	case SettingsOptionMarginMode:
		return h.handleMarginMode(ctx, userId, update, record)
	case SettingsOptionQuantityMode:
		return h.handleQuantityMode(ctx, userId, update, record)
	case SettingsOptionLeverage:
		return h.handleLeverage(ctx, userId, update, record)
	case SettingsOptionGridNum:
		return h.handleGridNum(ctx, userId, update, record)
	case SettingsOptionMarketSymbol:
		return h.handleMarketSymbol(ctx, userId, update, record)
	case SettingsOptionOrderSize:
		return h.handleOrderSize(ctx, userId, update, record)
	case SettingsOptionPriceLower:
		return h.handlePriceLower(ctx, userId, update, record)
	case SettingsOptionPriceUpper:
		return h.handlePriceUpper(ctx, userId, update, record)
	case SettingsOptionSlippage:
		return h.handleSlippage(ctx, userId, update, record)
	case SettingsOptionEnablePushNotification:
		return h.handleEnablePushNotificatione(ctx, userId, update, record)
	case SettingsOptionEnablePushMatchedNotification:
		return h.handleEnablePushMatchedNotification(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) refreshSettingsMessage(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Message != nil {
		chatId := update.Message.Chat.ID
		if update.Message.ReplyTo == nil {
			return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
		}

		route, ok := h.svcCtx.MessageCache.GetRoute(chatId, update.Message.ReplyTo.ID)
		if ok && route.Context != nil {
			return DisplayStrategSettings(ctx, h.svcCtx, userId, tele.Update{Message: route.Context}, record, false)
		}
		return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
	}

	return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
}

func (h *StrategySettingsHandler) handleGridMode(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Callback == nil {
		return nil
	}

	mode := strategy.ModeLong
	if record.Mode == strategy.ModeLong {
		mode = strategy.ModeShort
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateGridMode(ctx, record.ID, mode)
	if err == nil {
		record.Mode = mode
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[Mode]失败, %v", err)
	}

	chatId := util.ChatId(update.Callback.Message.Chat.ID)
	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)

	return h.refreshSettingsMessage(ctx, userId, update, record)
}

func (h *StrategySettingsHandler) handleMarginMode(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Callback == nil {
		return nil
	}

	mode := strategy.MarginModeCross
	if record.MarginMode == strategy.MarginModeCross {
		mode = strategy.MarginModeIsolated
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateMarginMode(ctx, record.ID, mode)
	if err == nil {
		record.MarginMode = mode
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[MarginMode]失败, %v", err)
	}

	chatId := util.ChatId(update.Callback.Message.Chat.ID)
	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)

	return h.refreshSettingsMessage(ctx, userId, update, record)
}

func (h *StrategySettingsHandler) handleQuantityMode(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Callback == nil {
		return nil
	}

	mode := strategy.QuantityModeArithmetic
	if record.QuantityMode == strategy.QuantityModeArithmetic {
		mode = strategy.QuantityModeGeometric
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateQuantityMode(ctx, record.ID, mode)
	if err == nil {
		record.QuantityMode = mode
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[QuantityMode]失败, %v", err)
	}

	chatId := util.ChatId(update.Callback.Message.Chat.ID)
	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chatId, text, 1)

	return h.refreshSettingsMessage(ctx, userId, update, record)
}

func (h *StrategySettingsHandler) handleLeverage(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写杠杆倍数，最高不得大于10。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionLeverage), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入金额
		chatId := update.Message.Chat.ID
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d < 1 || d > 10 {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效杠杆倍数(1~10)", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateLeverage(ctx, record.ID, d)
		if err == nil {
			record.Leverage = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[MaxLeverage]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleGridNum(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写网格数量，最高不得大于50。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionGridNum), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入金额
		chatId := update.Message.Chat.ID
		d, err := strconv.Atoi(update.Message.Text)
		if err != nil || d < 1 || d > 50 {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效网格数量(1~50)", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateGridNum(ctx, record.ID, d)
		if err == nil {
			record.GridNum = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[GridNum]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleMarketSymbol(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写交易代币Symbol，请确保交易平台已支持此币种。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionMarketSymbol), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入
		symbol := update.Message.Text
		chatId := update.Message.Chat.ID
		_, err := helper.GetMarketMetadata(ctx, h.svcCtx, record.Exchange, symbol)
		if err != nil {
			text := "❌ 交易平台不支持此币种，请检查后重试"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		if record.ExchangeApiKey != "" {
			result, err := h.svcCtx.StrategyModel.FindAllByExchangeAndAccountAndSymbol(ctx, record.Exchange, record.Account, symbol)
			if err != nil || len(result) > 0 {
				text := "❌ 同一交易账户不能创建多个相同币种的网格策略"
				util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
				return nil
			}
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateSymbol(ctx, record.ID, symbol)
		if err == nil {
			record.Symbol = symbol
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[Symbol]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleOrderSize(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if record.Exchange == "" || record.Symbol == "" {
		chat, ok := util.GetChat(update)
		if ok {
			text := "❌ 请先配置交易平台和交易币种"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		}
		return nil
	}

	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写单个网格交易代币的数量。"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionOrderSize), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入数量
		chatId := update.Message.Chat.ID
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入大于0的有效数字", 3)
			return nil
		}

		// 检查数量精度
		mm, err := helper.GetMarketMetadata(ctx, h.svcCtx, record.Exchange, record.Symbol)
		if err != nil {
			text := "❌ 交易平台不支持此币种，请检查后重试"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		if d.LessThan(mm.MinBaseAmount) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), fmt.Sprintf("❌ 代币数量不能小于%s", mm.MinBaseAmount), 3)
			return nil
		}

		if uint8(-d.Exponent()) > mm.SupportedSizeDecimals {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), fmt.Sprintf("❌ 代币数量小数位长度不能大于%d", mm.SupportedSizeDecimals), 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdateInitialOrderSize(ctx, record.ID, d)
		if err == nil {
			record.InitialOrderSize = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[InitialOrderSize]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handlePriceLower(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if record.Exchange == "" || record.Symbol == "" {
		chat, ok := util.GetChat(update)
		if ok {
			text := "❌ 请先配置交易平台和交易币种"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		}
		return nil
	}

	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写网格价格下限（单位: USD）\n\n💵 例: 100 → 代表100 USD"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionPriceLower), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入数量
		chatId := update.Message.Chat.ID
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入大于0的有效数字", 3)
			return nil
		}

		if d.GreaterThanOrEqual(record.PriceUpper) {
			text := "❌ 网格价格下限必须小于价格上限"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		// 检查数量精度
		mm, err := helper.GetMarketMetadata(ctx, h.svcCtx, record.Exchange, record.Symbol)
		if err != nil {
			text := "❌ 交易平台不支持此币种，请检查后重试"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		if uint8(-d.Exponent()) > mm.SupportedPriceDecimals {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), fmt.Sprintf("❌ 代币价格小数位长度不能大于%d", mm.SupportedPriceDecimals), 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdatePriceLower(ctx, record.ID, d)
		if err == nil {
			record.PriceLower = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[PriceLower]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handlePriceUpper(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if record.Exchange == "" || record.Symbol == "" {
		chat, ok := util.GetChat(update)
		if ok {
			text := "❌ 请先配置交易平台和交易币种"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		}
		return nil
	}

	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写网格价格上限（单位: USD）\n\n💵 例: 100 → 代表100 USD"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionPriceUpper), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入数量
		chatId := update.Message.Chat.ID
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThanOrEqual(decimal.Zero) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入大于0的有效数字", 3)
			return nil
		}

		if d.LessThanOrEqual(record.PriceLower) {
			text := "❌ 网格价格上限必须大于价格下限"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		// 检查数量精度
		mm, err := helper.GetMarketMetadata(ctx, h.svcCtx, record.Exchange, record.Symbol)
		if err != nil {
			text := "❌ 交易平台不支持此币种，请检查后重试"
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 3)
			return nil
		}

		if uint8(-d.Exponent()) > mm.SupportedPriceDecimals {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), fmt.Sprintf("❌ 代币价格小数位长度不能大于%d", mm.SupportedPriceDecimals), 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		err = h.svcCtx.StrategyModel.UpdatePriceUpper(ctx, record.ID, d)
		if err == nil {
			record.PriceUpper = d
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[PriceUpper]失败, %v", err)
		}
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleSlippage(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	// 步骤1
	if update.Callback != nil {
		chatId := update.Callback.Message.Chat.ID
		text := "🌳 填写市价交易的滑点百分比，清仓时将使用市价交易。\n\n🔢 例: 0.5 → 代表0.5%"
		msg, err := h.svcCtx.Bot.Send(util.ChatId(chatId), text, defaultSendOptions())
		if err != nil {
			logger.Debugf("[StrategySettingsHandler] 发送消息失败, %v", err)
			return err
		}

		route := cache.RouteInfo{Path: h.FormatPath(record.GUID, SettingsOptionSlippage), Context: update.Callback.Message}
		h.svcCtx.MessageCache.SetRoute(chatId, msg.ID, route)

		return nil
	}

	// 步骤2
	if update.Message != nil {
		deleteMessageAndReply(h.svcCtx.Bot, update.Message)

		// 检查输入金额
		chatId := update.Message.Chat.ID
		d, err := decimal.NewFromString(update.Message.Text)
		if err != nil || d.LessThan(decimal.Zero) || d.GreaterThan(decimal.NewFromInt(3)) {
			util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), "❌ 请输入有效百分比数字(0 <= slippage < 3)", 3)
			return nil
		}

		// 发送成功提示
		text := "✅ 配置修改成功"
		slippageBps := int(d.InexactFloat64() / 100 * 10000)
		err = h.svcCtx.StrategyModel.UpdateSlippageBps(ctx, record.ID, slippageBps)
		if err == nil {
			record.SlippageBps = &slippageBps
		} else {
			text = "❌ 配置修改失败, 请稍后重试"
			logger.Errorf("[StrategySettingsHandler] 更新配置[SlippageBps]失败, %v", err)
		}

		// 更新缓存数据
		strategyEngine, ok := GetStrategyEngine(ctx)
		if ok {
			strategyEngine.UpdateStrategy(record)
		}

		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

		return h.refreshSettingsMessage(ctx, userId, update, record)
	}

	return nil
}

func (h *StrategySettingsHandler) handleEnablePushNotificatione(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Callback == nil {
		return nil
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnablePushNotification(ctx, record.ID, !record.EnablePushNotification)
	if err == nil {
		record.EnablePushNotification = !record.EnablePushNotification
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnablePushNotification]失败, %v", err)
	}

	// 更新缓存数据
	strategyEngine, ok := GetStrategyEngine(ctx)
	if ok {
		strategyEngine.UpdateStrategy(record)
	}

	chatId := update.Callback.Message.Chat.ID
	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

	return h.refreshSettingsMessage(ctx, userId, update, record)
}

func (h *StrategySettingsHandler) handleEnablePushMatchedNotification(ctx context.Context, userId int64, update tele.Update, record *ent.Strategy) error {
	if update.Callback == nil {
		return nil
	}

	enablePushMatchedNotification := false
	if record.EnablePushMatchedNotification != nil && *record.EnablePushMatchedNotification {
		enablePushMatchedNotification = true
	}

	text := "✅ 配置修改成功"
	err := h.svcCtx.StrategyModel.UpdateEnablePushMatchedNotification(ctx, record.ID, !enablePushMatchedNotification)
	if err == nil {
		enablePushMatchedNotification = !enablePushMatchedNotification
		record.EnablePushMatchedNotification = &enablePushMatchedNotification
	} else {
		text = "❌ 配置修改失败, 请稍后重试"
		logger.Errorf("[StrategySettingsHandler] 更新配置[EnablePushMatchedNotification]失败, %v", err)
	}

	// 更新缓存数据
	strategyEngine, ok := GetStrategyEngine(ctx)
	if ok {
		strategyEngine.UpdateStrategy(record)
	}

	chatId := update.Callback.Message.Chat.ID
	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, util.ChatId(chatId), text, 1)

	return h.refreshSettingsMessage(ctx, userId, update, record)
}

func GenerateGridList(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) []decimal.Decimal {
	mm, err := helper.GetMarketMetadata(ctx, svcCtx, record.Exchange, record.Symbol)
	if err != nil {
		return nil
	}

	var prices []decimal.Decimal
	switch record.QuantityMode {
	case strategy.QuantityModeGeometric:
		prices, err = gridstrategy.GenerateGeometricGrid(record.PriceLower, record.PriceUpper, record.GridNum, int32(mm.SupportedPriceDecimals))
	case strategy.QuantityModeArithmetic:
		prices, err = gridstrategy.GenerateArithmeticGrid(record.PriceLower, record.PriceUpper, record.GridNum, int32(mm.SupportedPriceDecimals))
	}
	if err != nil {
		return nil
	}

	if record.Mode == strategy.ModeShort {
		slices.Reverse(prices)
	}

	return prices
}

func CalculateProfitMargin(record *ent.Strategy, prices []decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	if len(prices) < 2 {
		return decimal.Zero, decimal.Zero
	}

	profitMargin1, profitMargin2 := decimal.Zero, decimal.Zero
	p1, p2, p3, p4 := prices[0], prices[1], prices[len(prices)-2], prices[len(prices)-1]
	switch record.Mode {
	case strategy.ModeLong:
		profitMargin1 = p2.Sub(p1).Div(p1)
		profitMargin2 = p4.Sub(p3).Div(p3)
	case strategy.ModeShort:
		profitMargin1 = p1.Sub(p2).Div(p1)
		profitMargin2 = p3.Sub(p4).Div(p3)
	}

	if profitMargin1.LessThan(profitMargin2) {
		return profitMargin1, profitMargin2
	}
	return profitMargin2, profitMargin1
}

func DisplayStrategSettings(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy, newMessage bool) error {
	name := StrategyName(record)
	text := fmt.Sprintf("*%s* | 编辑策略 `%s`\n\n", svcCtx.Config.AppName, name)

	// 生成网格列表
	if record.Exchange != "" &&
		record.Symbol != "" &&
		record.GridNum > 0 &&
		record.PriceUpper.GreaterThan(decimal.Zero) &&
		record.PriceLower.GreaterThan(decimal.Zero) {

		var gridLabels []string
		totalInvestment := decimal.Zero
		prices := GenerateGridList(ctx, svcCtx, record)
		for idx, price := range prices {
			item := fmt.Sprintf("➖\\[ *%d* ] %s", idx, price)
			gridLabels = append(gridLabels, item)
			totalInvestment = totalInvestment.Add(record.InitialOrderSize.Mul(price))
		}

		// 截断网格列表
		if len(gridLabels) > MaxShowGridNum {
			n := MaxShowGridNum / 2
			part1 := lo.Slice(gridLabels, 0, n)
			part2 := lo.Slice(gridLabels, len(gridLabels)-n, len(gridLabels))
			gridLabels = make([]string, 0, len(gridLabels)+1)
			gridLabels = append(gridLabels, part1...)
			gridLabels = append(gridLabels, "➖   ... (省略中间网格)")
			gridLabels = append(gridLabels, part2...)
		}

		if len(gridLabels) > 0 {
			text += "网格列表:\n" + strings.Join(gridLabels, "\n")
		}

		if len(prices) > 2 {
			minProfitMargin, maxProfitMargin := CalculateProfitMargin(record, prices)
			minProfitMargin = minProfitMargin.Mul(decimal.NewFromInt(100)).Truncate(2)
			maxProfitMargin = maxProfitMargin.Mul(decimal.NewFromInt(100)).Truncate(2)
			if minProfitMargin.Equal(maxProfitMargin) {
				text += fmt.Sprintf("\n\n每格利润: *%v%%*", minProfitMargin)
			} else {
				text += fmt.Sprintf("\n\n每格利润: *%v%%* - *%v%%*", minProfitMargin, maxProfitMargin)
			}
			text += fmt.Sprintf("\n总投资额: %v USD", totalInvestment)
			text += fmt.Sprintf("\n初始保证金: %v USD", totalInvestment.Div(decimal.NewFromInt(int64(record.Leverage))).Truncate(2))
		}
	}

	connectStatus := "🔴"
	if testExchangeConnectivity(ctx, svcCtx, record) == nil {
		connectStatus = "🟢"
	}

	symbol := "未设置"
	if record.Symbol != "" {
		symbol = record.Symbol
	}

	orderSize := "未设置"
	if record.InitialOrderSize.GreaterThan(decimal.Zero) {
		orderSize = fmt.Sprintf("%s %s", record.InitialOrderSize, record.Symbol)
	}

	priceLower := "未设置"
	if record.PriceLower.GreaterThan(decimal.Zero) {
		priceLower = format.Price(record.PriceLower, 5)
	}

	priceUpper := "未设置"
	if record.PriceUpper.GreaterThan(decimal.Zero) {
		priceUpper = format.Price(record.PriceUpper, 5)
	}

	slippageBps := DefaultSlippageBps
	if record.SlippageBps != nil {
		slippageBps = *record.SlippageBps
	}

	h := StrategySettingsHandler{}
	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: fmt.Sprintf("%s 交易所: %s", connectStatus, lo.If(record.Exchange == "", "未设置").Else(record.Exchange)), Data: ExchangeSettingsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: fmt.Sprintf("杠杆倍数: %dX", record.Leverage), Data: h.FormatPath(record.GUID, SettingsOptionLeverage)},
				{Text: fmt.Sprintf("保证金: %s", lo.If(record.MarginMode == strategy.MarginModeCross, "全仓").Else("逐仓")), Data: h.FormatPath(record.GUID, SettingsOptionMarginMode)},
			},
			{
				{Text: fmt.Sprintf("交易币种: %s", symbol), Data: h.FormatPath(record.GUID, SettingsOptionMarketSymbol)},
				{Text: fmt.Sprintf("%s 网格模式: %s", lo.If(record.Mode == strategy.ModeLong, "🟢").Else("🔴"), lo.If(record.Mode == strategy.ModeLong, "做多").Else("做空")), Data: h.FormatPath(record.GUID, SettingsOptionGridMode)},
			},
			{
				{Text: fmt.Sprintf("网格数量: %d", record.GridNum), Data: h.FormatPath(record.GUID, SettingsOptionGridNum)},
				{Text: fmt.Sprintf("🔄 数量模式: %s", lo.If(record.QuantityMode == strategy.QuantityModeArithmetic, "等差").Else("等比")), Data: h.FormatPath(record.GUID, SettingsOptionQuantityMode)},
			},
			{
				{Text: fmt.Sprintf("🟰 单笔数量: %s", orderSize), Data: h.FormatPath(record.GUID, SettingsOptionOrderSize)},
			},
			{
				{Text: fmt.Sprintf("⬆️ 价格上限: %s", priceUpper), Data: h.FormatPath(record.GUID, SettingsOptionPriceUpper)},
			},
			{
				{Text: fmt.Sprintf("⬇️ 价格下限: %s", priceLower), Data: h.FormatPath(record.GUID, SettingsOptionPriceLower)},
			},
			{
				{Text: fmt.Sprintf("⚖️ 市价交易滑点: %v%%", float64(slippageBps)/10000*100.0), Data: h.FormatPath(record.GUID, SettingsOptionSlippage)},
			},
			{
				{Text: lo.If(record.EnablePushNotification, "🟢 开启成交通知").Else("🔴 关闭成交通知"), Data: h.FormatPath(record.GUID, SettingsOptionEnablePushNotification)},
				{Text: lo.If(record.EnablePushMatchedNotification != nil && *record.EnablePushMatchedNotification, "🟢 开启匹配通知").Else("🔴 关闭匹配通知"),
					Data: h.FormatPath(record.GUID, SettingsOptionEnablePushMatchedNotification)},
			},
			{
				{Text: "◀️ 返回上级", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
				{Text: "⏪ 返回主页", Data: "/home"},
			},
		},
	}

	_, err := util.ReplyMessage(svcCtx.Bot, update, text, replyMarkup, newMessage)
	if err != nil {
		logger.Debugf("[DisplayStrategSettings] 生成UI失败, %v", err)
	}
	return nil
}
