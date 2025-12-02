package handler

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/exchange/paradex"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/fachebot/omni-grid-bot/internal/util/format"
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

	return DisplayStrategyDetailsWithStrategyGUID(ctx, h.svcCtx, userId, update, guid)
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

func marketSymbol(record *ent.Strategy) string {
	if record.Symbol == "" {
		return "未设置"
	}

	switch record.Exchange {
	case exchange.Lighter:
		return fmt.Sprintf("[%s](https://app.lighter.xyz/trade/%s)", record.Symbol, record.Symbol)
	case exchange.Paradex:
		return fmt.Sprintf("[%s](https://app.paradex.trade/trade/%s)", record.Symbol, paradex.FormatUsdPerpMarket(record.Symbol))
	case exchange.Variational:
		return fmt.Sprintf("[%s](https://omni.variational.io/perpetual/%s)", record.Symbol, record.Symbol)
	default:
		return "未设置"
	}
}

func StrategyDetailsText(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) string {
	name := StrategyName(record)
	text := fmt.Sprintf("*%s* | 策略详情 `%s`\n\n", svcCtx.Config.AppName, name)

	// 查询网格列表
	grids, err := svcCtx.GridModel.FindAllByStrategyIdOrderAsc(ctx, record.GUID)
	if err != nil {
		logger.Errorf("[StrategyDetailsText] 查询网格列表失败, id: %s, %v", record.GUID, err)
	}
	grids = lo.Filter(grids, func(item *ent.Grid, idx int) bool {
		return item.BuyClientOrderId != nil || item.SellClientOrderId != nil
	})

	totalInvestment := decimal.Zero
	for _, lvl := range grids {
		totalInvestment = totalInvestment.Add(lvl.Quantity.Mul(lvl.Price))
	}

	// 账户信息
	text += "📊 账户\n"
	exchangeAccount := lo.If(record.Exchange != "", record.Exchange).Else("未设置")
	if record.Exchange != "" && record.Account != "" {
		if len(record.Account) < 10 {
			exchangeAccount += "#" + record.Account
		} else {
			exchangeAccount += "#" + record.Account[:6] + "..." + record.Account[len(record.Account)-4:]
		}
	}
	text += fmt.Sprintf("┣ 交易平台: *%s*\n", exchangeAccount)

	var position *exchange.Position
	var availableBalance decimal.Decimal
	if record.Exchange != "" && record.Account != "" {
		account, err := helper.GetAccountInfo(ctx, svcCtx, record)
		if err == nil {
			availableBalance = account.AvailableBalance
			positionSide := lo.If(record.Mode == strategy.ModeLong, exchange.PositionSideLong).Else(exchange.PositionSideShort)
			position, _ = lo.Find(account.Positions, func(item *exchange.Position) bool {
				return record.Symbol == item.Symbol && item.Side == positionSide
			})
		}
	}
	text += fmt.Sprintf("┗ 可用余额: `%s` USD\n\n", availableBalance)

	// 策略信息
	if record.Status != strategy.StatusActive || record.StartTime == nil {
		text += "📌 策略\n"
	} else {
		text += fmt.Sprintf("📌 策略(%s)\n", time.Since(*record.StartTime))
	}
	positionSide := lo.If(record.Mode == strategy.ModeLong, "🟢做多").Else("🔴做空")
	marginMode := lo.If(record.MarginMode == strategy.MarginModeCross, "全仓").Else("逐仓")
	text += fmt.Sprintf("┣ 方向: %s | 杠杆: **%dX** | %s\n", positionSide, record.Leverage, marginMode)
	text += fmt.Sprintf("┣ 交易标的: %s\n", marketSymbol(record))
	text += fmt.Sprintf("┣ 价格区间: %s\n", lo.If(record.PriceLower.IsZero() || record.PriceUpper.IsZero(), "未设置").
		Else(fmt.Sprintf("$%s ~ $%s", record.PriceLower, record.PriceUpper)))
	text += fmt.Sprintf("┗ 单格投入: %s\n\n", lo.If(record.Symbol != "" && !record.InitialOrderSize.IsZero(), fmt.Sprintf("%s %s", record.InitialOrderSize, record.Symbol)).Else("未设置"))

	// 查询最新价格
	lastPrice := decimal.Zero
	if record.Exchange != "" && record.Symbol != "" {
		lastPrice, err = helper.GetLastTradePrice(ctx, svcCtx, record.Exchange, record.Symbol)
		if err != nil {
			logger.Debugf("[StrategyDetailsText] 查询最新价格失败, exchange: %s, symbol: %s, %v", record.Exchange, record.Symbol, err)
		}
	}

	// 持仓信息
	if position != nil {
		text += "📦 持仓\n"
		text += fmt.Sprintf("┣ 持仓数量: %s %s\n", position.Position, position.Symbol)
		text += fmt.Sprintf("┣ 持仓价值: $%s\n", format.Price(position.Position.Mul(lastPrice), 5))
		text += fmt.Sprintf("┣ 强平价格: *$%s*\n", format.Price(position.LiquidationPrice, 5))
		text += fmt.Sprintf("┗ 平均持仓成本: $%s\n\n", format.Price(position.AvgEntryPrice, 5))
	}

	// 计算未实现收益
	unrealizedPnl := decimal.Zero
	switch record.Mode {
	case strategy.ModeLong:
		size, cost, err := svcCtx.MatchedTradeModel.QueryOpeLongPositionAndCost(ctx, record.GUID)
		if err == nil {
			unrealizedPnl = size.Mul(lastPrice).Sub(cost)
		} else {
			logger.Errorf("[StrategyDetailsText] 查询未平多仓和成本失败, id: %s, %v", record.GUID, err)
		}
	case strategy.ModeShort:
		size, cost, err := svcCtx.MatchedTradeModel.QueryOpenShortPositionAndCost(ctx, record.GUID)
		if err == nil {
			unrealizedPnl = cost.Sub(size.Mul(lastPrice))
		} else {
			logger.Errorf("[StrategyDetailsText] 查询未平多仓和成本失败, id: %s, %v", record.GUID, err)
		}
	}

	// 收益信息
	realizedPnl, err := svcCtx.MatchedTradeModel.QueryTotalProfit(ctx, record.GUID)
	if err != nil {
		logger.Warnf("[StrategyDetailsText] 查询已实现利润失败, id: %s, %v", record.GUID, err)
	}
	pnl := realizedPnl.Add(unrealizedPnl)
	if position != nil {
		pnl = pnl.Sub(position.TotalFundingPaidOut)
	}

	text += "💰 收益\n"
	if record.Status == strategy.StatusActive && record.StartTime != nil && totalInvestment.GreaterThan(decimal.Zero) {
		days := decimal.NewFromFloat(float64(time.Since(*record.StartTime)) / float64(time.Hour*24))
		apr := pnl.Div(totalInvestment).Div(days).Mul(decimal.NewFromInt(365)).Mul(decimal.NewFromInt(100))
		text += fmt.Sprintf("┣ 总利润: %s(*APR %v%%*)\n", pnl.Truncate(5), apr.Truncate(4))
	} else {
		text += fmt.Sprintf("┣ 总利润: %s\n", pnl.Truncate(5))
	}
	text += fmt.Sprintf("┣ 已实现利润: %s\n", realizedPnl.Truncate(5))
	text += fmt.Sprintf("┗ 未实现利润: %s\n\n", unrealizedPnl.Truncate(5))

	// 显示网格挂单
	if len(grids) == 0 {
		text += fmt.Sprintf("➖[💵] *当前价格*: $*%s*\n\n", lastPrice)
	} else {
		gridList := formatGridListWithCurrentPrice(lastPrice, grids)
		if record.Mode == strategy.ModeLong {
			slices.Reverse(gridList)
		}
		text += "🟢 买入订单 | 🔴 卖出订单\n\n" + strings.Join(gridList, "\n")
		text += fmt.Sprintf("\n\n总投资额: $%v\n", totalInvestment)
		text += fmt.Sprintf("初始保证金: $%v\n\n", totalInvestment.Div(decimal.NewFromInt(int64(record.Leverage))).Truncate(2))
	}

	text += fmt.Sprintf("🕒 更新时间: [%s]\n\n⚠️ 重要提示:\n▸ *停止策略会清空之前的网格记录!*", util.FormaTime(time.Now()))
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
				{Text: "💎 一键平仓", Data: ClosePositionHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🔄 刷新界面", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
				{Text: "🗒 匹配记录", Data: MatchedTradesHandler{}.FormatPath(record.GUID, 1)},
			},
			{
				{Text: "⚙️ 编辑策略", Data: StrategySettingsHandler{}.FormatPath(record.GUID)},
				{Text: "🗑 删除策略", Data: DeleteStrategyHandler{}.FormatPath(record.GUID)},
			},
			{

				{Text: "🚫 关闭页面", Data: DeleteMessageHandler{}.FormatPath()},
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

func DisplayStrategyDetailsWithStrategyGUID(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, guid string) error {
	record, err := svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, svcCtx, userId, update, 1)
		}
		logger.Errorf("[StrategyDetailsHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return DisplayStrategyList(ctx, svcCtx, userId, update, 1)
	}

	return DisplayStrategyDetails(ctx, svcCtx, userId, update, record)
}
