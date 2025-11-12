package handler

import (
	"context"
	"errors"

	"github.com/fachebot/omni-grid-bot/internal/engine"
	"github.com/fachebot/omni-grid-bot/internal/ent"
	"github.com/fachebot/omni-grid-bot/internal/exchange"
	"github.com/fachebot/omni-grid-bot/internal/helper"
	"github.com/fachebot/omni-grid-bot/internal/svc"
	"github.com/fachebot/omni-grid-bot/internal/util"
	tele "gopkg.in/telebot.v4"
)

func StrategyName(record *ent.Strategy) string {
	return record.GUID[len(record.GUID)-4:]
}

func GetStrategyEngine(ctx context.Context) (*engine.StrategyEngine, bool) {
	v := ctx.Value("engine")
	if v == nil {
		return nil, false
	}

	engine, ok := v.(*engine.StrategyEngine)
	if !ok {
		return nil, false
	}

	return engine, true
}

func defaultSendOptions() *tele.SendOptions {
	return &tele.SendOptions{
		ParseMode:             tele.ModeMarkdown,
		ReplyMarkup:           &tele.ReplyMarkup{ForceReply: true},
		DisableWebPagePreview: true,
	}
}

func deleteMessageAndReply(bot *tele.Bot, message *tele.Message) {
	deleteMessages := []*tele.Message{message}
	if message.ReplyTo != nil {
		deleteMessages = append(deleteMessages, message.ReplyTo)
	}
	util.DeleteMessages(bot, deleteMessages, 0)
}

func testLighterConnectivity(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) error {
	lighterClient, err := helper.GetLighterClient(svcCtx, record)
	if err != nil {
		return err
	}

	_, err = lighterClient.GetAccountInactiveOrders(ctx, "", 1)
	return err
}

func testExchangeConnectivity(ctx context.Context, svcCtx *svc.ServiceContext, record *ent.Strategy) error {
	if record.Exchange == "" {
		return errors.New("exchange not configured")
	}

	switch record.Exchange {
	case exchange.Lighter:
		return testLighterConnectivity(ctx, svcCtx, record)
	default:
		return errors.New("exchange unsupported")
	}
}
