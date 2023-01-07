package services

import (
	"fmt"

	"github.com/bots-empire/base-bot/msgs"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/model"
)

const topUsersByBalance = 3

func (u *Users) GetTopUsersByBalance(countOfUsers int) ([]*model.User, int, error) {
	//get top 3 users by balance
	if countOfUsers != 0 {
		users, err := u.GetUsers(topUsersByBalance)
		if err != nil {
			return nil, countOfUsers, err
		}

		if len(users) < 3 {
			return nil, countOfUsers, errors.New("Top players < 3, no users in bot")
		}

		return users, countOfUsers, nil
	}

	//get 10 percent users by balance
	countOfUsers = u.admin.CountUsers()
	if countOfUsers > 30 {
		countOfUsers /= 10
	}

	users, err := u.GetUsers(countOfUsers)
	if err != nil {
		return nil, countOfUsers, err
	}

	if len(users) < 3 {
		return nil, countOfUsers, errors.New("Top players < 3, no users in bot")
	}

	return users, countOfUsers, nil
}

func (u *Users) topShowedPlayersFromMainText(numberOfTop int, users []*model.User) string {
	text := u.bot.LangText(u.bot.LanguageInBot[0], "top_3_players_main",
		numberOfTop+1,
		users[numberOfTop].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[numberOfTop],
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[0],
		users[0].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[1],
		users[1].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[2],
		users[2].Balance,
	)

	return text
}

func (u *Users) ShowedTopPlayersTextAndMarkup(topNumber int, users []*model.User) (string, *tgbotapi.InlineKeyboardMarkup) {
	text := u.bot.LangText(u.bot.LanguageInBot[0], "top_3_players",
		topNumber+1,
		users[topNumber].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[topNumber],
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[0],
		users[0].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[1],
		users[1].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[2],
		users[2].Balance,
	)
	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlDataButton("get_reward", "/get_reward"))).
		Build(u.bot.Language[u.bot.BotLang])

	return text, &markUp
}

func (u *Users) topPlayersText(users []*model.User, i int) string {
	text := u.bot.LangText(u.bot.LanguageInBot[0], "top_players",
		i+1,
		users[0].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[0],
		users[1].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[1],
		users[2].Balance,
		model.AdminSettings.GlobalParameters[u.bot.LanguageInBot[0]].Parameters.TopReward[2],
		users[i].Balance,
		i,
		users[i-1].Balance,
	)

	return text
}

func (u *Users) GetRewardCommand(s *model.Situation) error {
	var userNum int
	top, err := u.GetTop()
	if err != nil {
		return err
	}

	for i := range top {
		if top[i].UserID == s.User.ID {
			userNum = i
		}
	}

	balance, err := u.GetUserBalanceFromID(s.User.ID)
	if err != nil {
		return err
	}

	err = u.UpdateTop3Balance(s.User.ID,
		balance+model.AdminSettings.GlobalParameters[s.BotLang].Parameters.TopReward[userNum])
	if err != nil {
		return err
	}

	err = u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, nil, u.bot.LangText(
		u.bot.LanguageInBot[0],
		"top_3_players_reward_taken",
		userNum+1,
		s.User.Balance,
		model.AdminSettings.GlobalParameters[u.bot.BotLang].Parameters.TopReward[0],
		top[0].Balance,
		model.AdminSettings.GlobalParameters[u.bot.BotLang].Parameters.TopReward[1],
		top[1].Balance,
		model.AdminSettings.GlobalParameters[u.bot.BotLang].Parameters.TopReward[2],
		top[2].Balance,
	))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to edit markup msgID = %d", s.CallbackQuery.Message.MessageID))
	}

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, nil, u.bot.LangText(u.bot.LanguageInBot[0], "got_reward"))
}

func (u *Users) SendTopUsersByBalance(id int64, markUp *tgbotapi.InlineKeyboardMarkup, text string) error {
	return u.Msgs.NewParseMarkUpMessage(id, &markUp, text)
}
