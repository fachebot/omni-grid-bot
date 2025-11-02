package handler

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/strategy"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/samber/lo"
	tele "gopkg.in/telebot.v4"
)

type StrategyDetailsHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategyDetailsHandler(svcCtx *svc.ServiceContext) *StrategyDetailsHandler {
	return &StrategyDetailsHandler{svcCtx: svcCtx}
}

func (h StrategyDetailsHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/details/%s", guid)
}

func (h *StrategyDetailsHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/details/{uuid}", h.handle)
}

func (h *StrategyDetailsHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[StrategyDetailsHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
	}

	return DisplayStrategyDetails(ctx, h.svcCtx, userId, update, record)
}

func StrategyDetailsText(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy) string {
	name := StrategyName(record)
	text := fmt.Sprintf("*Lighter网格策略* | 策略详情 `%s`\n\n", name)

	text += fmt.Sprintf("📊 交易平台: *%s*\n", lo.If(record.Exchange != "", record.Exchange).Else("未设置"))
	text += fmt.Sprintf("📈 交易标的: %s\n", lo.If(record.Symbol != "", record.Symbol).Else("未设置"))
	text += fmt.Sprintf("🔢 杠杆倍数: %dX\n", record.Leverage)
	text += fmt.Sprintf("🔒 保证金模式: %s\n", lo.If(record.MarginMode == strategy.MarginModeCross, "全仓").Else("逐仓"))
	text += fmt.Sprintf("📈 价格区间: %s\n", lo.If(record.PriceLower.IsZero() || record.PriceUpper.IsZero(), "未设置").
		Else(fmt.Sprintf("$%s ~ $%s", record.PriceLower, record.PriceUpper)))
	text += fmt.Sprintf("⚙️ 单格投入: %s\n", lo.If(record.Symbol != "" && !record.InitialOrderSize.IsZero(), fmt.Sprintf("%s %s", record.InitialOrderSize, record.Symbol)).Else("未设置"))
	text += "💵 总利润: 0\n"
	text += "✅ 已实现利润: 0\n"
	text += "❓ 未实现利润: 0\n"

	return text
}

func DisplayStrategyDetails(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy) error {
	status := "🟢 策略运行中"
	if record.Status == strategy.StatusInactive {
		status = "🔴 策略已停止"
	}

	text := StrategyDetailsText(ctx, svcCtx, userId, update, record)

	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: status, Data: CompletedTradesHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🔄 刷新界面", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
				{Text: "🗒 完成记录", Data: CompletedTradesHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "⚙️ 编辑策略", Data: StrategySettingsHandler{}.FormatPath(record.GUID)},
				{Text: "🗑 删除策略", Data: DeleteStrategyHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "◀️ 返回列表", Data: StrategyListHandler{}.FormatPath(1)},
			},
		},
	}

	_, err := util.ReplyMessage(svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[DisplayStrategyDetails] 生成UI失败, %v", err)
	}
	return nil
}
