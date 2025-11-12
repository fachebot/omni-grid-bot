package handler

import (
	"context"
	"fmt"
	"strconv"

	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	tele "gopkg.in/telebot.v4"
)

type StrategyListHandler struct {
	svcCtx *svc.ServiceContext
}

func NewStrategyListHandler(svcCtx *svc.ServiceContext) *StrategyListHandler {
	return &StrategyListHandler{svcCtx: svcCtx}
}

func (h StrategyListHandler) FormatPath(page int) string {
	return fmt.Sprintf("/strategy/list/%d", page)
}

func (h *StrategyListHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/list", h.handle)
	router.HandleFunc("/strategy/list/{page:[0-9]+}", h.handle)
}

func (h *StrategyListHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
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

	err := DisplayStrategyList(ctx, h.svcCtx, userId, update, page)
	if err != nil {
		logger.Debugf("[StrategyListHandler] 处理UI失败, %v", err)
	}

	return nil
}

func DisplayStrategyList(ctx context.Context, svcCtx *svc.ServiceContext, userId int64, update tele.Update, page int) error {
	if page < 1 {
		return nil
	}

	// 查询策略列表
	const limit = 10
	offset := (page - 1) * limit
	userStrategyList, total, err := svcCtx.StrategyModel.FindAllByOwner(ctx, userId, offset, limit)
	if err != nil {
		return err
	}

	totalPage := total / limit
	if total%limit != 0 {
		totalPage += 1
	}

	if page > totalPage {
		page = totalPage
		offset := (page - 1) * limit
		userStrategyList, total, err = svcCtx.StrategyModel.FindAllByOwner(ctx, userId, offset, limit)
		if err != nil {
			return err
		}
	}

	// 生成策略列表
	var inlineKeyboard [][]tele.InlineButton
	for _, item := range userStrategyList {
		status := "🟢"
		if item.Status != strategy.StatusActive {
			status = "🔴"
		}

		label := "未完成初始化"
		if item.Exchange != "" && item.Symbol != "" {
			label = fmt.Sprintf("%s | %s | %s", item.Exchange, item.Symbol, item.Mode)
		}

		name := StrategyName(item)
		inlineKeyboard = append(inlineKeyboard, []tele.InlineButton{
			{Text: fmt.Sprintf("%s %s | %s", status, name, label), Data: StrategyDetailsHandler{}.FormatPath(item.GUID)},
		})
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

	inlineKeyboard = append(inlineKeyboard, pageButtons)
	inlineKeyboard = append(inlineKeyboard, []tele.InlineButton{
		{Text: "🔄 刷新界面", Data: StrategyListHandler{}.FormatPath(1)},
		{Text: "➕ 创建策略", Data: CreateStrategyHandler{}.FormatPath()},
	})

	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: inlineKeyboard,
	}
	text := fmt.Sprintf("*%s | 专业的合约网格机器人*\n\n", svcCtx.Config.AppName)
	text += "`盈利如春雨, 润物无声, 渐丰收!`\n\n"

	text += "🔥 为震荡而生：在横盘和波动市中自动低买高卖。\n"
	text += "⏳ 全托管执行：挂单、补仓、止盈、止损全自动。\n"
	text += "\n[Lighter](app.lighter.xyz/?referral=28913X95) | [ParaDex](https://app.paradex.trade/r/resilienthunter70)"

	_, err = util.ReplyMessage(svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[DisplayStrategyList] 生成UI失败, %v", err)
	}
	return nil
}
