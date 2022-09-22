package auth

import (
	"github.com/bots-empire/base-bot/msgs"

	"github.com/bots-empire/shazam-bot/internal/model"
)

type Auth struct {
	bot *model.GlobalBot

	msgs *msgs.Service
}

func NewAuthService(bot *model.GlobalBot, msgs *msgs.Service) *Auth {
	return &Auth{
		bot:  bot,
		msgs: msgs,
	}
}
