package telebot

import (
	"context"
	"fmt"

	"github.com/fachebot/perp-dex-grid-bot/internal/logger"
	"github.com/fachebot/perp-dex-grid-bot/internal/svc"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/handler"
	"github.com/fachebot/perp-dex-grid-bot/internal/telebot/pathrouter"
	"github.com/fachebot/perp-dex-grid-bot/internal/util"

	tele "gopkg.in/telebot.v4"
)

type TeleBot struct {
	ctx    context.Context
	cancel context.CancelFunc
	router *pathrouter.Router
	svcCtx *svc.ServiceContext
}

func NewTeleBot(svcCtx *svc.ServiceContext) *TeleBot {
	ctx, cancel := context.WithCancel(context.Background())
	b := &TeleBot{
		ctx:    ctx,
		cancel: cancel,
		svcCtx: svcCtx,
		router: pathrouter.NewRouter(),
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
	replyMarkup := &tele.ReplyMarkup{
		InlineKeyboard: [][]tele.InlineButton{
			{
				{Text: "🎯 我的跟单", Data: "/1"},
				{Text: "📢 钱包监控", Data: "/2"},
			},
		},
	}

	chat, _ := util.GetChat(update)
	text := "*HyperCopier* | 专注 Hyperliquid 聪明钱跟单"
	text = text + "\n\n通过实时追踪 [Hyperliquid](https://hyperliquid.xyz/) 高胜率或高收益地址，将其开平仓行为转化为可参数化的复制策略：你可自定义仓位比例、最大杠杆、风控阈值、止盈止损与黑白名单，实现精细化自动交易体验。"
	text = text + fmt.Sprintf("\n\n👤 UID: `%d`\n💳 身份: *普通会员*\n", chat.ID)
	text = text + "\n\n发现聪明钱 👉[HyperX](https://hyper.faster100x.com/hyperliquid/wallet-discover?ref=HYPERCOPIER)"
	_, err := util.ReplyMessage(b.svcCtx.Bot, update, text, replyMarkup)
	if err != nil {
		logger.Debugf("[TeleBot] 处理主页失败, %v", err)
	}

	return nil
}

func (b *TeleBot) handleUpdate(c tele.Context) error {
	// 获取用户ID
	update := c.Update()
	chat, ok := util.GetChat(update)
	if !ok {
		return nil
	}

	logger.Debugf("[TeleBot] 收到新消息, chat: %d, username: <%s>, title: %s, type: %s",
		chat.ID, chat.Username, chat.Title, chat.Type)

	// 私聊消息
	if chat.Type == tele.ChatPrivate {
		// 处理文本消息
		if update.Message != nil {
			if update.Message.Text == "/start" {
				err := b.handleHome(update)
				if err != nil {
					logger.Debugf("[TeleBot] 处理主页失败, %v", err)
				}
				return nil
			}

			if update.Message.ReplyTo != nil {
				chatId := update.Message.ReplyTo.Chat.ID
				messageID := update.Message.ReplyTo.ID
				route, ok := b.svcCtx.MessageCache.GetRoute(chatId, messageID)
				if ok {
					err := b.router.Execute(b.ctx, route.Path, chat.ID, update)
					if err != nil {
						logger.Debugf("[TeleBot] 处理路由失败, path: %s, %v", route.Path, err)
					}
				}
			}

			return nil
		}

		// 处理回调查询
		if update.Callback != nil {
			err := b.router.Execute(b.ctx, update.Callback.Data, chat.ID, update)
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
