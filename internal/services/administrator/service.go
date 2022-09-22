package administrator

import (
	"github.com/bots-empire/base-bot/mailing"
	"github.com/bots-empire/base-bot/msgs"

	"github.com/bots-empire/shazam-bot/internal/model"
)

type Admin struct {
	bot *model.GlobalBot

	mailing *mailing.Service
	msgs    *msgs.Service
}

func NewAdminService(bot *model.GlobalBot, mailing *mailing.Service, msgs *msgs.Service) *Admin {
	return &Admin{
		bot:     bot,
		mailing: mailing,
		msgs:    msgs,
	}
}
