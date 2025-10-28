package util

import (
	"errors"
	"strconv"
	"time"

	"github.com/fachebot/perp-dex-grid-bot/internal/logger"

	tele "gopkg.in/telebot.v4"
)

type ChatId int64

func (chatId ChatId) Recipient() string {
	return strconv.FormatInt(int64(chatId), 10)
}

func GetChat(update tele.Update) (*tele.Chat, bool) {
	if update.Message != nil {
		return update.Message.Chat, true
	}

	if update.EditedMessage != nil {
		return update.EditedMessage.Chat, true
	}

	if update.Callback != nil {
		return update.Callback.Message.Chat, true
	}

	if update.ChannelPost != nil {
		return update.ChannelPost.Chat, true
	}

	if update.EditedChannelPost != nil {
		return update.EditedChannelPost.Chat, true
	}

	return nil, false
}

func ReplyMessage(bot *tele.Bot, update tele.Update, text string, replyMarkup *tele.ReplyMarkup) (*tele.Message, error) {
	var message *tele.Message
	if update.Message != nil {
		message = update.Message
	} else if update.Callback != nil {
		message = update.Callback.Message
	} else {
		return nil, errors.New("unsupported update type")
	}

	if message.Sender.ID == bot.Me.ID {
		if message.Caption != "" {
			return bot.EditCaption(message, text, &tele.SendOptions{
				ParseMode:             tele.ModeMarkdown,
				ReplyMarkup:           replyMarkup,
				DisableWebPagePreview: true,
			})
		}
		return bot.Edit(message, text, &tele.SendOptions{
			ParseMode:             tele.ModeMarkdown,
			ReplyMarkup:           replyMarkup,
			DisableWebPagePreview: true,
		})
	}

	photo := &tele.Photo{File: tele.FromURL("https://pub-7d8b66050cd845cfa208b34a0b2dd62a.r2.dev/lighter.jpg")}
	photo.Caption = text
	return bot.Send(message.Chat, photo, &tele.SendOptions{
		ParseMode:             tele.ModeMarkdown,
		ReplyMarkup:           replyMarkup,
		DisableWebPagePreview: true,
	})
}

func DeleteMessages(bot *tele.Bot, messages []*tele.Message, delaySeconds int) {
	delFunc := func() {
		for _, msg := range messages {
			if err := bot.Delete(msg); err != nil {
				logger.Debugf("[TeleBot] 删除消息失败, %v", err)
			}
		}
	}

	if delaySeconds <= 0 {
		delFunc()
	} else {
		time.AfterFunc(time.Second*time.Duration(delaySeconds), delFunc)
	}
}

func SendMarkdownMessage(bot *tele.Bot, recipient tele.Recipient, text string, replyMarkup *tele.ReplyMarkup) (*tele.Message, error) {
	return bot.Send(recipient, text, &tele.SendOptions{
		ParseMode:             tele.ModeMarkdown,
		DisableWebPagePreview: true,
		ReplyMarkup:           replyMarkup,
	})
}

func SendMarkdownMessageAndDelayDeletion(bot *tele.Bot, recipient tele.Recipient, text string, delaySeconds int) {
	msg, err := SendMarkdownMessage(bot, recipient, text, nil)
	if err == nil {
		DeleteMessages(bot, []*tele.Message{msg}, delaySeconds)
	} else {
		logger.Debugf("[TeleBot] 发送消息失败, chatId: %s, text: %s, %v", recipient.Recipient(), text, err)
	}
}
