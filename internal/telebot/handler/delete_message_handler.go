package handler

import (
	"context"

	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"
	tele "gopkg.in/telebot.v4"
)

type DeleteMessageHandler struct {
	svcCtx *svc.ServiceContext
}

func NewDeleteMessageHandler(svcCtx *svc.ServiceContext) *DeleteMessageHandler {
	return &DeleteMessageHandler{svcCtx: svcCtx}
}

func (h DeleteMessageHandler) FormatPath() string {
	return "/message/delete"
}

func (h *DeleteMessageHandler) AddRouter(router *pathrouter.Router) {
	router.HandleFunc("/message/delete", h.handle)
}

func (h *DeleteMessageHandler) handle(ctx context.Context, vars map[string]string, userId int64, update tele.Update) error {
	if update.Message != nil {
		util.DeleteMessages(h.svcCtx.Bot, []*tele.Message{update.Message}, 0)
	} else if update.EditedMessage != nil {
		util.DeleteMessages(h.svcCtx.Bot, []*tele.Message{update.EditedMessage}, 0)
	} else if update.Callback != nil {
		util.DeleteMessages(h.svcCtx.Bot, []*tele.Message{update.Callback.Message}, 0)
	}

	return nil
}
