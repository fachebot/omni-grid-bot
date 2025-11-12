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

type DeleteStrategyHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeleteStrategyHandler(svcCtx *svc.ServiceContext) *DeleteStrategyHandler {
	return &DeleteStrategyHandler{svcCtx: svcCtx}
}

func (h DeleteStrategyHandler) FormatPath(guid string) string {
	return fmt.Sprintf("/strategy/delete/%s", guid)
}

func (h *DeleteStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/delete/{uuid}", h.handle)
	router.HandleFunc("/strategy/delete/{uuid}/{confirm}", h.handle)
}

func (h *DeleteStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
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
		logger.Errorf("[DeleteStrategyHandler] 查询策略失败, id: %s, %v", guid, err)
		return nil
	}

	chat, ok := util.GetChat(update)
	if !ok || record.Owner != userId {
		return DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
	}

	// 检查策略状态
	if record.Status == strategy.StatusActive {
		util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, "❌ 删除策略之前, 请先停止策略", 1)
		return nil
	}

	// 显示删除菜单
	_, confirm := vars["confirm"]
	if !confirm {
		inlineKeyboard := [][]tele.InlineButton{
			{
				{Text: "🔴 删除策略", Data: h.FormatPath(guid) + "/confirm"},
			},
			{
				{Text: "◀️ 返回上级", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🟣 我点错了", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
			},
			{
				{Text: "🟢 取消删除", Data: StrategyDetailsHandler{}.FormatPath(record.GUID)},
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

	// 执行删除策略
	text := fmt.Sprintf("✅ *%s* 策略删除成功", StrategyName(record))
	err = h.svcCtx.StrategyModel.Delete(ctx, record.ID)
	if err != nil {
		text = fmt.Sprintf("❌ *%s* 策略删除失败, 请稍后再试", StrategyName(record))
		logger.Errorf("[DeleteStrategyHandler] 删除策略失败, id: %d, %v", record.ID, err)
	} else {
		err = DisplayStrategyList(ctx, h.svcCtx, userId, update, 1)
		if err != nil {
			logger.Warnf("[DeleteStrategyHandler] 处理UI失败, %v", err)
		}
	}

	util.SendMarkdownMessageAndDelayDeletion(h.svcCtx.Bot, chat, text, 1)

	return nil
}
