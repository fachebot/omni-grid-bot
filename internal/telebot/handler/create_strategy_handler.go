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
	"github.com/google/uuid"
	tele "gopkg.in/telebot.v4"
)

type CreateStrategyHandler struct {
	svcCtx *svc.ServiceContext
}

func NewCreateStrategyHandler(svcCtx *svc.ServiceContext) *CreateStrategyHandler {
	return &CreateStrategyHandler{svcCtx: svcCtx}
}

func (h CreateStrategyHandler) FormatPath() string {
	return "/strategy/create"
}

func (h *CreateStrategyHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/strategy/create", h.handle)
}

func (h *CreateStrategyHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	guid, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	record := ent.Strategy{
		GUID:         guid.String(),
		Owner:        userId,
		Mode:         strategy.ModeLong,
		MarginMode:   strategy.MarginModeCross,
		Leverage:     2,
		QuantityMode: strategy.QuantityModeArithmetic,
		GridNum:      50,
		Status:       strategy.StatusInactive,
	}
	_, err = h.svcCtx.StrategyModel.Save(ctx, record)
	if err != nil {
		logger.Errorf("[CreateStrategyHandler] 保存策略信息失败, %v", err)
		return err
	}

	text := fmt.Sprintf("✅ *%s* 策略创建成功", StrategyName(&record))
	util.SendMarkdownMessage(h.svcCtx.Bot, chat, text, nil)

	DisplayStrategSettings(ctx, h.svcCtx, userId, update, &record, true)

	return nil
}
