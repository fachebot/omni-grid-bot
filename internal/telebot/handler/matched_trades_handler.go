package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	"github.com/fachebot/omni-grid-bot/internal/util/format"
	tele "gopkg.in/telebot.v4"
)

type MatchedTradesHandler struct {
	svcCtx *svc.ServiceContext
}

func NewMatchedTradesHandler(svcCtx *svc.ServiceContext) *MatchedTradesHandler {
	return &MatchedTradesHandler{svcCtx: svcCtx}
}

func (h MatchedTradesHandler) FormatPath(guid string, page int) string {
	return fmt.Sprintf("/strategy/trades/%s/%d", guid, page)
}

func (h *MatchedTradesHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/trades/{uuid}/{page:[0-9]+}", h.handle)
}

func (h *MatchedTradesHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	var page int
	val, ok := vars["page"]
	if !ok {
		page = 1
	} else {
		n, err := strconv.Atoi((val))
		if err != nil {
			page = 1
		} else {
			page = n
		}
	}

	// 查询策略信息
	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[MatchedTradesHandler] 查询策略信息失败, id: %s, %v", guid, err)
		return nil
	}

	if record.Owner != userId {
		return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
	}

	// 查询交易记录
	const limit = 10
	offset := (page - 1) * limit
	trades, total, err := h.svcCtx.MatchedTradeModel.FinAllMatchedTrades(ctx, guid, offset, limit)
	if err != nil {
		logger.Errorf("[MatchedTradesHandler] 查询策略匹配交易列表失败, userId: %d, strategy: %s, %v", userId, guid, err)
		return nil
	}

	totalPage := total / limit
	if total%limit != 0 {
		totalPage += 1
	}

	if page > totalPage {
		page = totalPage
		offset := (page - 1) * limit
		trades, total, err = h.svcCtx.MatchedTradeModel.FinAllMatchedTrades(ctx, guid, offset, limit)
		if err != nil {
			logger.Errorf("[MatchedTradesHandler] 查询策略匹配交易列表失败, userId: %d, strategy: %s, %v", userId, guid, err)
			return nil
		}
	}

	// 生成交易列表
	items := make([]string, 0)
	for _, trade := range trades {
		if trade.Profit == nil ||
			trade.BuyBaseAmount == nil ||
			trade.SellBaseAmount == nil {
			continue
		}

		buyPrice := format.Price(trade.BuyQuoteAmount.Div(*trade.BuyBaseAmount), 5)
		sellPrice := format.Price(trade.SellQuoteAmount.Div(*trade.SellBaseAmount), 5)

		switch record.Mode {
		case strategy.ModeLong:
			date := util.FormaDate(time.Unix(*trade.SellOrderTimestamp, 0))
			s := fmt.Sprintf("*%s* 🟢 开多 %s %s, 价格 %s, 平多价格 %s, 利润 %v USD", date, trade.BuyBaseAmount, trade.Symbol, buyPrice, sellPrice, *trade.Profit)
			items = append(items, s)
		case strategy.ModeShort:
			date := util.FormaDate(time.Unix(*trade.BuyOrderTimestamp, 0))
			s := fmt.Sprintf("*%s* 🔴 开空 %s %s, 价格 %s, 平空价格 %s, 利润 %v USD", date, trade.SellBaseAmount, trade.Symbol, sellPrice, buyPrice, *trade.Profit)
			items = append(items, s)
		}
	}

	// 多页翻页功能
	var pageButtons []tele.InlineButton
	if total > limit {
		nextPage := page + 1
		previousPage := page - 1
		if previousPage < 1 {
			page = 1
			previousPage = 0
		}
		if nextPage > totalPage {
			page = totalPage
			nextPage = 0
		}
		pageButtons = []tele.InlineButton{
			{Text: "⬅️ 上一页", Data: StrategyListHandler{}.FormatPath(previousPage)},
			{Text: fmt.Sprintf("%d/%d", page, totalPage), Data: StrategyListHandler{}.FormatPath(0)},
			{Text: "➡️ 下一页", Data: StrategyListHandler{}.FormatPath(nextPage)},
		}
	}

	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			pageButtons,
			{
				{Text: "◀️ 返回上级", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
				{Text: "⏪ 返回主页", Data: StrategyListHandler{}.FormatPath(1)},
			},
		},
	}

	name := StrategyName(record)
	text := fmt.Sprintf("*%s* | 匹配记录 `%s`\n\n", h.svcCtx.Config.AppName, name)
	if len(items) == 0 {
		text += "暂无网格匹配记录"
	} else {
		text += fmt.Sprintf("匹配交易次数: *%d*\n\n", total) + strings.Join(items, "\n\n")
	}

	_, err = util.ReplyMessage(h.svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[MatchedTradesHandler] 生成UI失败, %v", err)
	}
	return nil
}
