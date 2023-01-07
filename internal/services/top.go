package services

import (
	"github.com/bots-empire/shazam-bot/internal/model"
)

func (u *Users) TopListPlayerCommand(s *model.Situation) error {
	users, _, err := u.GetTopUsersByBalance(topUsersByBalance)
	if err != nil {
		return err
	}

	err = u.SendTopListForUser(s, users)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) SendTopListForUser(s *model.Situation, users []*model.User) error {
	for i, val := range users {
		if s.User.ID == val.ID {
			text := u.topShowedPlayersFromMainText(i, users)
			return u.Msgs.NewParseMessage(s.User.ID, text)
		}

		count := u.admin.CountUsers()

		allUsers, err := u.GetUsers(count)
		if err != nil {
			return err
		}

		for j, value := range allUsers {
			if value.ID == s.User.ID {
				text := u.topPlayersText(allUsers, j)
				return u.Msgs.NewParseMessage(s.User.ID, text)
			}
		}

	}

	return nil
}
