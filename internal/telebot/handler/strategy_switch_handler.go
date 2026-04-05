package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/omni-grid-bot/internal/engine"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/model"
	gridstrategy "github.com/fachebot/omni-grid-bot/internal/strategy"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/shopspring/decimal"
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

func (h StrategySwitchHandler) FormatStopPath(guid string, stopType StopType) string {
	return fmt.Sprintf("/strategy/switch/%s/%s", guid, stopType)
}

func (h *StrategySwitchHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/switch/{uuid}", h.handle)
	router.HandleFunc("/strategy/switch/{uuid}/{stop}", h.handle)
}

func (h *StrategySwitchHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[StrategySwitchHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return nil
	}

	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	strategyEngine, ok := GetStrategyEngine(ctx)
	if !ok {
		return nil
	}

	// 用户全局锁
	userLock := h.svcCtx.GetUserLock(userId)
	if !userLock.TryLock() {
		text := "❌ 您有其他操作正在处理中，请稍后再试"
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		return nil
	}
	defer userLock.Unlock()

	// 选择开关操作
	stopType, ok := vars["stop"]
	if !ok {
		if record.Status == strategy.StatusActive {
			text := StrategyDetailsText(ctx, h.svcCtx, record)
			inlineKeyboard := [][]tele.InlineButton{
				{
					{Text: "❌ 取消关闭", Data: StrategyDetailsHandler{}.FormatPath(guid)},
				},
				{
					{Text: "1️⃣ 仅关闭策略", Data: h.FormatStopPath(guid, StopTypeStop)},
				},
				{
					{Text: "2️⃣ 关闭并平仓", Data: h.FormatStopPath(guid, StopTypeClose)},
				},
			}

			replyMarkup := &tele.ReplyMarkup{
				InlineKeyboard: inlineKeyboard,
			}
			_, err := util.ReplyMessage(h.svcCtx.Bot, update, text, replyMarkup)
			return err
		}

		if record.Status == strategy.StatusInactive {
			return h.handleStartStrategy(ctx, userId, update, record, strategyEngine)
		}
		return nil
	}

	// 处理关闭策略
	switch StopType(stopType) {
	case StopTypeStop:
		return h.handleStopStrategy(ctx, userId, update, record, strategyEngine)
	case StopTypeClose:
		return h.handleStopStrategyAndClose(ctx, userId, update, record, strategyEngine)
	}

	return nil
}

func (h *StrategySwitchHandler) handleStartStrategy(
	ctx context.Context, userId int64, update tele.Update, record *ent.Strategy, strategyEngine *engine.StrategyEngine,
) error {
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	msg, err := util.SendMarkdownMessage(h.svcCtx.Bot, chat, "🚀 正在开启策略...", nil)
	if err != nil {
		return err
	}

	// 检查策略状态
	if record.Status != strategy.StatusActive {
		text := "❌ 策略正在运行中"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 测试交易所连接
	account, err := helper.GetAccountInfo(ctx, h.svcCtx, record)
	if err != nil {
		text := "❌ 连接交易平台失败，请检查交易平台配置"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 查询币种信息
	if record.Symbol == "" {
		text := "❌ 此策略没有配置交易币种，请检查配置后重试"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}
	mm, err := helper.GetMarketMetadata(ctx, h.svcCtx, record.Exchange, record.Symbol)
	if err != nil {
		text := "❌ 交易平台不支持此币种，请检查配置后重试"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 检查单笔数量
	if record.InitialOrderSize.LessThan(mm.MinBaseAmount) {
		text := fmt.Sprintf("❌ 代币数量不能小于%s", mm.MinBaseAmount)
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	if uint8(-record.InitialOrderSize.Exponent()) > mm.SupportedSizeDecimals {
		text := fmt.Sprintf("❌ 代币数量小数位长度不能大于%d", mm.SupportedSizeDecimals)
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 检查交易金额
	if record.InitialOrderSize.Mul(record.PriceLower).LessThan(mm.MinQuoteAmount) {
		text := fmt.Sprintf("❌ 单笔交易金额不能小于 %s USD，请调整单笔数量和价格下限", mm.MinQuoteAmount)
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 检查网格策略
	result, err := h.svcCtx.StrategyModel.FindAllByExchangeAndAccountAndSymbol(ctx, record.Exchange, record.Account, record.Symbol)
	if err != nil || len(result) > 1 {
		text := "❌ 同一交易账户不能创建多个相同币种的网格策略"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 生成网格价格
	prices, err := gridstrategy.GenerateGridPrices(record, mm.SupportedPriceDecimals)
	if err != nil || len(prices) == 0 {
		text := "❌ 生成网格失败，请调整价格上下区间后重试"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 校验保证金数量
	positionValue := decimal.Zero
	maxPositionValue := account.AvailableBalance.Mul(decimal.NewFromInt(int64(record.Leverage)))
	for _, price := range prices {
		positionValue = positionValue.Add(price.Mul(record.InitialOrderSize))
	}
	if positionValue.GreaterThanOrEqual(maxPositionValue) {
		text := fmt.Sprintf("❌ 账户保证金余额不足，必须大于 %s USD，请充值后重试", positionValue.Div(decimal.NewFromInt(int64(record.Leverage))).Truncate(2))
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 检查入场价格
	if record.EntryPrice != nil && record.EntryPrice.GreaterThan(decimal.Zero) {
		if record.EntryPrice.LessThan(record.PriceLower) || record.EntryPrice.GreaterThan(record.PriceUpper) {
			text := "❌ 策略入场价格必须在价格区间内，请检查配置后重试"
			_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
			return err
		}
	}

	// 检查 Rate Limit 配额
	if record.Exchange == exchange.Lighter {
		status := h.svcCtx.LighterClient.RateLimiter().Status()
		if status.RemainingRequests < 10 {
			text := "⏳ 等待 Rate Limit 配额..."
			util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		}
	}

	// 初始化网格策略
	err = gridstrategy.InitGridStrategy(ctx, h.svcCtx, record, prices)
	if err != nil {
		logger.Warnf("[StrategySwitchHandler] 初始化网格策略失败, id: %s, symbol: %s, %v", record.GUID, record.Symbol, err)

		text := fmt.Sprintf("❌ 初始化网格策略失败，请检查配置后重试\n\n`%s`", err.Error())
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}
	record.Status = strategy.StatusActive

	// 开始运行策略
	err = strategyEngine.StartStrategy(gridstrategy.NewGridStrategy(h.svcCtx, strategyEngine, record))
	if err != nil {
		logger.Warnf("[StrategySwitchHandler] 运行策略失败, id: %s, symbol: %s, %v", record.GUID, record.Symbol, err)

		text := "❌ 运行策略失败，请联系管理员"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, "✅ 策略已开启", nil)

	return DisplayStrategyDetails(ctx, h.svcCtx, userId, update, record)
}

func (h *StrategySwitchHandler) handleStopStrategy(
	ctx context.Context, userId int64, update tele.Update, record *ent.Strategy, strategyEngine *engine.StrategyEngine,
) error {
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	msg, err := util.SendMarkdownMessage(h.svcCtx.Bot, chat, "🛑 正在关闭策略...", nil)
	if err != nil {
		return err
	}

	// 检查策略状态
	if record.Status != strategy.StatusActive {
		text := "❌ 策略未运行"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}

	// 停止网格策略
	strategyEngine.StopStrategy(record.GUID)

	// 检查 Rate Limit 配额
	if record.Exchange == exchange.Lighter {
		status := h.svcCtx.LighterClient.RateLimiter().Status()
		if status.RemainingRequests < 5 {
			text := "⏳ 等待 Rate Limit 配额..."
			util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		}
	}

	// 取消用户订单
	name := util.StrategyName(record)
	err = CancelAllOrders(ctx, h.svcCtx, record)
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 取消用户订单失败, id: %s, exchange: %s, account: %s, symbol: %s, %v",
			record.GUID, record.Exchange, record.Account, record.Symbol, err)

		text := fmt.Sprintf("⚠️ [%s]取消网格 *%s* %s 订单失败", name, record.Symbol, record.Mode)
		util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
	}

	// 更新策略状态
	err = util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewMatchedTradeModel(tx.MatchedTrade).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatus(ctx, record.ID, strategy.StatusInactive)
	})
	if err != nil {
		logger.Errorf("[StrategySwitchHandler] 更新策略状态失败, guid: %s, %v", record.GUID, err)

		text := "❌ 关闭策略失败，请稍后再试"
		_, err = util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, text, nil)
		return err
	}
	record.Status = strategy.StatusInactive

	util.ReplyMessage(h.svcCtx.Bot, tele.Update{Message: msg}, "✅ 策略已停止", nil)

	return DisplayStrategyDetails(ctx, h.svcCtx, userId, update, record)
}

func (h *StrategySwitchHandler) handleStopStrategyAndClose(
	ctx context.Context, userId int64, update tele.Update, record *ent.Strategy, strategyEngine *engine.StrategyEngine,
) error {
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	msg, _ := util.ReplyMessage(h.svcCtx.Bot, update, "🛑 正在关闭并平仓...", nil)

	// 检查策略状态
	if record.Status != strategy.StatusActive {
		h.svcCtx.Bot.Edit(msg, "❌ 策略未运行", nil)
		return nil
	}

	// 检查 Rate Limit 配额
	if record.Exchange == exchange.Lighter {
		status := h.svcCtx.LighterClient.RateLimiter().Status()
		if status.RemainingRequests < 5 {
			h.svcCtx.Bot.Edit(msg, "⏳ 等待 Rate Limit 配额...", nil)
		}
	}

	// 取消用户订单
	name := util.StrategyName(record)
	err := CancelAllOrders(ctx, h.svcCtx, record)
	if err != nil {
		text := fmt.Sprintf("⚠️ [%s]取消网格 *%s* %s 订单失败", name, record.Symbol, record.Mode)
		util.SendMarkdownMessage(h.svcCtx.Bot, chat, text, nil)
		logger.Errorf("[StrategySwitchHandler] 取消用户订单失败, id: %s, exchange: %s, account: %s, symbol: %s, %v",
			record.GUID, record.Exchange, record.Account, record.Symbol, err)
	}

	// 停止网格策略
	strategyEngine.StopStrategy(record.GUID)

	// 关闭用户仓位
	err = ClosePosition(ctx, h.svcCtx, record)
	if err != nil {
		text := fmt.Sprintf("⚠️ [%s]关闭网格 *%s* %s 仓位失败", name, record.Symbol, record.Mode)
		util.SendMarkdownMessage(h.svcCtx.Bot, chat, text, nil)
		logger.Errorf("[StrategySwitchHandler] 关闭网格仓位失败, id: %s, exchange: %s, account: %s, symbol: %s, %v",
			record.GUID, record.Exchange, record.Account, record.Symbol, err)
	}

	// 更新策略状态
	err = util.Tx(ctx, h.svcCtx.DbClient, func(tx *ent.Tx) error {
		err = model.NewGridModel(tx.Grid).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		err = model.NewMatchedTradeModel(tx.MatchedTrade).DeleteByStrategyId(ctx, record.GUID)
		if err != nil {
			return err
		}

		return model.NewStrategyModel(tx.Strategy).UpdateStatus(ctx, record.ID, strategy.StatusInactive)
	})
	if err != nil {
		text := "❌ 关闭策略失败，请稍后再试"
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		logger.Errorf("[StrategySwitchHandler] 更新策略状态失败, guid: %s, %v", record.GUID, err)
		return nil
	}
	record.Status = strategy.StatusInactive

	h.svcCtx.Bot.Edit(msg, "✅ 策略已停止", nil)

	return DisplayStrategyDetails(ctx, h.svcCtx, userId, update, record)
}
