package services

import (
	"encoding/json"
	"fmt"
	"runtime/debug"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/log"
)

var (
	panicLogger = log.NewDefaultLogger().Prefix("panic cather")
)

func (u *Users) panicCather(update *tgbotapi.Update) {
	msg := recover()
	if msg == nil {
		return
	}

	panicText := fmt.Sprintf("%s\npanic in backend: message = %s\n%s",
		u.bot.BotLink,
		msg,
		string(debug.Stack()),
	)
	panicLogger.Warn(panicText)

	u.Msgs.SendNotificationToDeveloper(panicText, false)

	data, err := json.MarshalIndent(update, "", "  ")
	if err != nil {
		return
	}

	u.Msgs.SendNotificationToDeveloper(string(data), false)
}
