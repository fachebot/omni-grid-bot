package handler

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fachebot/perp-dex-grid-bot/internal/ent"
	"github.com/fachebot/perp-dex-grid-bot/internal/ent/strategy"
	"github.com/fachebot/perp-dex-grid-bot/internal/helper"
	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
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

func formatGridLevelDisplay(lvl *ent.Grid) string {
	text := fmt.Sprintf("➖\\[ *%d* ] %s ", lvl.Level, lvl.Price)
	if lvl.BuyClientOrderId != nil {
		text += "🟢"
	}
	if lvl.SellClientOrderId != nil {
		text += "🔴"
	}
	return text
}

func formatGridListWithCurrentPrice(lastPrice decimal.Decimal, grids []*ent.Grid) []string {
	if len(grids) == 0 {
		return nil
	}

	// 查找当前位置
	pos := -1
	for idx, lvl := range grids {
		if lvl.Price.GreaterThanOrEqual(lastPrice) {
			break
		}
		pos = idx
	}

	half := MaxShowGridNum / 2
	left := lo.Slice(grids, 0, pos+1)
	right := lo.Slice(grids, pos+1, len(grids))

	// 生成左边部分
	gridLabels := make([]string, 0, MaxShowGridNum)
	if len(left) > 0 {
		n := half
		if len(right) == 0 {
			n = MaxShowGridNum
		}

		if len(left) > n {
			first := left[0]
			gridLabels = append(gridLabels, formatGridLevelDisplay(first))
			gridLabels = append(gridLabels, "➖   ... (省略中间网格)")

			left = left[len(left)-n:]
		}

		for _, lvl := range left {
			gridLabels = append(gridLabels, formatGridLevelDisplay(lvl))
		}
	}

	gridLabels = append(gridLabels, fmt.Sprintf("➖[💵] *当前价格*: $*%s*", lastPrice))

	// 生成右边部分
	if len(right) > 0 {
		n := half
		if len(left) == 0 {
			n = MaxShowGridNum
		}

		last := right[len(right)-1]
		if len(right) > n {
			right = right[:n]
		}

		for _, lvl := range right {
			gridLabels = append(gridLabels, formatGridLevelDisplay(lvl))
		}

		if last != right[len(right)-1] {
			gridLabels = append(gridLabels, "➖   ... (省略中间网格)")
			gridLabels = append(gridLabels, formatGridLevelDisplay(last))
		}
	}

	return gridLabels
}

func StrategyDetailsText(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) string {
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

	if record.Status == strategy.StatusActive {
		// 查询最新价格
		lastPrice, err := helper.GetLastTradePrice(ctx, svcCtx, record.Exchange, record.Symbol)
		if err != nil {
			logger.Debugf("[StrategyDetailsText] 查询最新价格失败, exchange: %s, symbol: %s, %v", record.Exchange, record.Symbol, err)
		}

		// 查询网格列表
		grids, err := svcCtx.GridModel.FindAllByStrategyIdOrderAsc(ctx, record.GUID)
		if err != nil {
			logger.Errorf("[StrategyDetailsText] 查询网格列表失败, id: %s, %v", record.GUID, err)
		}
		grids = lo.Filter(grids, func(item *ent.Grid, idx int) bool {
			return item.BuyClientOrderId != nil || item.SellClientOrderId != nil
		})

		totalInvestment := decimal.Zero
		if len(grids) > 0 {
			for _, lvl := range grids {
				totalInvestment = totalInvestment.Add(lvl.Quantity.Mul(lvl.Price))
			}
			gridList := formatGridListWithCurrentPrice(lastPrice, grids)
			if record.Mode == strategy.ModeLong {
				slices.Reverse(gridList)
			}
			text += "\n🟢 买入订单 | 🔴 卖出订单\n\n" + strings.Join(gridList, "\n")
			text += fmt.Sprintf("\n\n总投资额: %v USD", totalInvestment)
			text += fmt.Sprintf("\n初始保证金: %v USD", totalInvestment.Div(decimal.NewFromInt(int64(record.Leverage))).Truncate(2))
		}
	}

	text += fmt.Sprintf("\n\n🕒 更新时间: [%s]\n\n⚠️ 重要提示:\n▸ *停止策略会清空之前的网格记录!*", util.FormaTime(time.Now()))
	return text
}

func DisplayStrategyDetails(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, record *ent.Strategy) error {
	status := "🟢 策略运行中"
	if record.Status == strategy.StatusInactive {
		status = "🔴 策略已停止"
	}

	text := StrategyDetailsText(ctx, svcCtx, record)

	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: status, Data: StrategySwitchHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🔄 刷新界面", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
				{Text: "🗒 匹配记录", Data: CompletedTradesHandler{}.FormatPath(record.GUID)},
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
