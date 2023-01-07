package services

import (
	"fmt"

	"github.com/bots-empire/shazam-bot/internal/model"
)

func (u *Users) TopForMailing() {
	defer u.panicCather(nil)
	if !model.AdminSettings.GlobalParameters[u.bot.BotLang].Parameters.TopMailing {
		return
	}

	users, countOfUsers, err := u.GetTopUsersByBalance(0)
	if err != nil {
		u.Msgs.SendNotificationToDeveloper(fmt.Sprintf("count of users/10 in bot %d\nerr: %s",
			countOfUsers, err.Error()), false)
	}

	err = u.SendMailingTop(users, countOfUsers)
	if err != nil {
		u.Msgs.SendNotificationToDeveloper(fmt.Sprintf("failed to send mailing top\ncount of users/10 in bot %d\nerr: %s",
			countOfUsers, err.Error()), false)
	}
}

func (u *Users) SendMailingTop(users []*model.User, countUsers int) error {
	//Send top 3 markup message
	for i := 0; i < topUsersByBalance; i++ {
		text, markUp := u.ShowedTopPlayersTextAndMarkup(i, users)

		err := u.SendTopUsersByBalance(users[i].ID, markUp, text)
		if err != nil {
			return err
		}
	}

	//send top > 3 message
	for i := topUsersByBalance; i <= countUsers; i++ {
		text := u.topPlayersText(users, i)

		err := u.SendTopUsersByBalance(users[i].ID, nil, text)
		if err != nil {
			return err
		}
	}

	return nil
}
