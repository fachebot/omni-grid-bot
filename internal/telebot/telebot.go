package telebot

import (
	"context"
	"strings"

	"github.com/fachebot/omni-grid-bot/internal/engine"
	"github.com/fachebot/omni-grid-bot/internal/logger"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/telebot/handler"
	"github.com/fachebot/omni-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/omni-grid-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

type TeleBot struct {
	ctx            context.Context
	cancel         context.CancelFunc
	router         *pathrouter.Router
	svcCtx         *svc.ServiceContext
	strategyEngine *engine.StrategyEngine
}

func NewTeleBot(svcCtx *svc.ServiceContext, strategyEngine *engine.StrategyEngine) *TeleBot {
	ctx, cancel := context.WithCancel(context.Background())
	b := &TeleBot{
		ctx:            ctx,
		cancel:         cancel,
		svcCtx:         svcCtx,
		strategyEngine: strategyEngine,
		router:         pathrouter.NewRouter(),
	}
	b.initRoutes()
	return b
}

func (b *TeleBot) Stop() {
	logger.Infof("[TeleBot] 准备停止服务")
	b.cancel()
	b.svcCtx.Bot.Stop()
	logger.Infof("[TeleBot] 服务已经停止")
}

func (b *TeleBot) Start() {
	logger.Infof("[TeleBot] 开始运行服务")

	h := func(c tele.Context) error {
		return b.handleUpdate(c)
	}
	b.svcCtx.Bot.Handle(tele.OnText, h)
	b.svcCtx.Bot.Handle(tele.OnEdited, h)
	b.svcCtx.Bot.Handle(tele.OnQuery, h)
	b.svcCtx.Bot.Handle(tele.OnCallback, h)
	b.svcCtx.Bot.Handle(tele.OnChannelPost, h)

	go b.svcCtx.Bot.Start()
}

func (b *TeleBot) initRoutes() {
	b.router.HandleFunc("/home", func(
		ctx context.Context,
		vars map[string]string,
		userId int64,
		update tele.Update,
	) error {
		return b.handleHome(update)
	})

	handler.InitRoutes(b.svcCtx, b.router)
}

func (b *TeleBot) handleHome(update tele.Update) error {
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}
	return handler.DisplayStrategyList(b.ctx, b.svcCtx, chat.ID, update, 1)
}

func (b *TeleBot) handleUpdate(c tele.Context) error {
	// 获取用户ID
	update := c.Update()
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	ctx := context.WithValue(b.ctx, handler.ContextKeyEngine, b.strategyEngine)

	logger.Debugf("[TeleBot] 收到新消息, chat: %d, username: <%s>, title: %s, type: %s",
		chat.ID, chat.Username, chat.Title, chat.Type)

	// 私聊消息
	if chat.Type == tele.ChatPrivate {
		if !b.svcCtx.Config.TelegramBot.IsWhiteListUser(chat.ID) {
			util.SendMarkdownMessage(b.svcCtx.Bot, chat, "🚫 非白名单用户, 不允许使用此机器人", nil)
			return nil
		}

		// 处理文本消息
		if update.Message != nil {
			if strings.HasPrefix(update.Message.Text, "/start") {
				if update.Message.Payload == "" {
					err := b.handleHome(update)
					if err != nil {
						logger.Debugf("[TeleBot] 处理主页失败, %v", err)
					}
					return nil
				}

				util.DeleteMessages(b.svcCtx.Bot, []*tele.Message{update.Message}, 0)

				return handler.DisplayStrategyDetailsWithStrategyGUID(
					ctx, b.svcCtx, chat.ID, update, update.Message.Payload)
			}

			if update.Message.ReplyTo != nil {
				chatId := update.Message.ReplyTo.Chat.ID
				messageID := update.Message.ReplyTo.ID
				route, ok := b.svcCtx.MessageCache.GetRoute(chatId, messageID)
				if ok {
					err := b.router.Execute(ctx, route.Path, chat.ID, update)
					if err != nil {
						logger.Debugf("[TeleBot] 处理路由失败, path: %s, %v", route.Path, err)
					}
				}
			}

			return nil
		}

		// 处理回调查询
		if update.Callback != nil {
			err := b.router.Execute(ctx, update.Callback.Data, chat.ID, update)
			if err == nil {
				if err = c.Respond(); err != nil {
					logger.Debugf("[TeleBot] 应答 CallbackQuery 失败, id: %s, %v", update.Callback.ID, err)
				}
			} else {
				logger.Errorf("[TeleBot] 处理 CallbackQuery 失败, %v", err)
				if err = c.RespondAlert("操作失败, 请稍后再试"); err != nil {
					logger.Debugf("[TeleBot] 应答 CallbackQuery 失败, id: %s, %v", update.Callback.ID, err)
				}
			}
		}
	}

	return nil
}
