package services

import (
	"github.com/bots-empire/base-bot/msgs"

	"github.com/bots-empire/shazam-bot/internal/model"
	"github.com/bots-empire/shazam-bot/internal/services/administrator"
	"github.com/bots-empire/shazam-bot/internal/services/auth"
)

type Users struct {
	bot *model.GlobalBot

	auth  *auth.Auth
	admin *administrator.Admin
	Msgs  *msgs.Service
}

func NewUsersService(bot *model.GlobalBot, auth *auth.Auth, admin *administrator.Admin, msgs *msgs.Service) *Users {
	return &Users{
		bot:   bot,
		auth:  auth,
		admin: admin,
		Msgs:  msgs,
	}
}

func (u *Users) GelBotLang() string {
	return u.bot.BotLang
}
