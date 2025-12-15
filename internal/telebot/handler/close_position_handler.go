package handler

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/ent/strategy"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	tele "gopkg.in/telebot.v4"
)

type ClosePositionHandler struct {
	svcCtx *svc.ServiceContext
}

func NewClosePositionHandler(svcCtx *svc.ServiceContext) *ClosePositionHandler {
	return &ClosePositionHandler{svcCtx: svcCtx}
}

func (h ClosePositionHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/position/close/%s", guid)
}

func (h *ClosePositionHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/position/close/{uuid}", h.handle)
	router.HandleFunc("/position/close/{uuid}/{confirm}", h.handle)
}

func (h *ClosePositionHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, ok := vars["uuid"]
	if !ok {
		return nil
	}

	// 查询策略信息
	record, err := h.svcCtx.StrategyModel.FindOneByGUID(ctx, guid)
	if err != nil {
		if ent.IsNotFound(err) {
			return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		}
		logger.Errorf("[ClosePositionHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	chat, ok := util.GetChat(update)
	if !ok || record.Owner != userId {
		return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
	}

	// 检查策略状态
	if record.Status == strategy.StatusActive {
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, "❌ 平仓之前请先停止策略", 1)
		return nil
	}

	// 测试交易所连接
	err = testExchangeConnectivity(ctx, h.svcCtx, record)
	if err != nil {
		text := "❌ 连接交易平台失败，请检查交易平台配置"
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 3)
		return nil
	}

	// 显示平仓菜单
	_, confirm := vars["confirm"]
	if !confirm {
		inlineKeyboard := [][]tele.InlineButton{
			{
				{Text: "🔴 确认平仓", Data: h.FormatPath(guid) + "/confirm"},
			},
			{
				{Text: "◀️ 返回上级", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🟣 我点错了", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🟢 取消平仓", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
			},
		}

		rand.Shuffle(len(inlineKeyboard), func(i, j int) {
			inlineKeyboard[i], inlineKeyboard[j] = inlineKeyboard[j], inlineKeyboard[i]
		})

		replyMarkup := &tele.ReplyMarkup{
			InlineKeyboard: inlineKeyboard,
		}
		text := StrategyDetailsText(ctx, h.svcCtx, record)
		_, err := util.ReplyMessage(h.svcCtx.Bot, update, text, replyMarkup)
		return err
	}

	// 执行平仓操作
	text := fmt.Sprintf("✅ *%s* 平仓成功", util.StrategyName(record))
	err = ClosePosition(ctx, h.svcCtx, record)
	if err != nil {
		text = fmt.Sprintf("❌ *%s* 平仓失败, 请稍后再试", util.StrategyName(record))
		logger.Errorf("[ClosePositionHandler] 平仓失败, id: %d, %v", record.ID, err)
	} else {
		err = DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		if err != nil {
			logger.Warnf("[ClosePositionHandler] 处理UI失败, %v", err)
		}
	}

	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 1)

	return nil
}
