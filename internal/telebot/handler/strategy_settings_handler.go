package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/strategy"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/fachebot/perp-dex-grid-bot/internal/util/format"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	tele "gopkg.in/telebot.v4"
)

type SettingsOption int

var (
	SettingsOptionMaxLeverage      SettingsOption = 1
	SettingsOptionGridNum          SettingsOption = 2
	SettingsOptionGridMode         SettingsOption = 3
	SettingsOptionMarginMode       SettingsOption = 4
	SettingsOptionQuantityMode     SettingsOption = 5
	SettingsOptionOrderSize        SettingsOption = 6
	SettingsOptionPriceLower       SettingsOption = 7
	SettingsOptionPriceUpper       SettingsOption = 8
	SettingsOptionExchangeSettings SettingsOption = 9
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

	switch SettingsOption(optionValue) {
	case SettingsOptionGridMode:
		return h.handleGridMode(ctx, userId, update, record)
	case SettingsOptionMarginMode:
		return h.handleMarginMode(ctx, userId, update, record)
	case SettingsOptionQuantityMode:
		return h.handleQuantityMode(ctx, userId, update, record)
	}

	return nil
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

	return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
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

	return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
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

	return DisplayStrategSettings(ctx, h.svcCtx, userId, update, record, false)
}

func DisplayStrategSettings(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy, newMessage bool) error {
	name := StrategyName(record)
	text := fmt.Sprintf("*Lighter网格策略* | 编辑策略 `%s`\n\n", name)

	connectStatus := "🔴"
	if testExchangeConnectivity(ctx, svcCtx, record) == nil {
		connectStatus = "🟢"
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

	h := StrategySettingsHandler{}
	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: fmt.Sprintf("%s 交易所: %s", connectStatus, lo.If(record.Exchange == "", "未设置").Else(record.Exchange)), Data: ExchangeSettingsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: fmt.Sprintf("杠杆倍数: %dX", record.Leverage), Data: "/"},
				{Text: fmt.Sprintf("保证金: %s", lo.If(record.MarginMode == strategy.MarginModeCross, "全仓").Else("逐仓")), Data: h.FormatPath(record.GUID, SettingsOptionMarginMode)},
			},
			{
				{Text: "交易币种: BTC", Data: "/"},
				{Text: fmt.Sprintf("%s 网格模式: %s", lo.If(record.Mode == strategy.ModeLong, "🟢").Else("🔴"), lo.If(record.Mode == strategy.ModeLong, "做多").Else("做空")), Data: h.FormatPath(record.GUID, SettingsOptionGridMode)},
			},
			{
				{Text: fmt.Sprintf("网格数量: %d", record.GridNum), Data: "/"},
				{Text: fmt.Sprintf("🔄 数量模式: %s", lo.If(record.QuantityMode == strategy.QuantityModeArithmetic, "等差").Else("等比")), Data: h.FormatPath(record.GUID, SettingsOptionQuantityMode)},
			},
			{
				{Text: fmt.Sprintf("🟰 单笔数量: %s", orderSize), Data: "/"},
			},
			{
				{Text: fmt.Sprintf("⬇️ 价格下限: %s", priceLower), Data: "/"},
			},
			{
				{Text: fmt.Sprintf("⬆️ 价格上限: %s", priceUpper), Data: "/"},
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
