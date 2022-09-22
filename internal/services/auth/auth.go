package auth

import (
	"database/sql"
	"math/rand"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	model2 "github.com/bots-empire/shazam-bot/internal/model"
	"github.com/bots-empire/shazam-bot/internal/services/administrator"
)

const (
	//typeFriend = "friend"
	//typeGroup  = "group"

	getUsersUserQuery = "SELECT * FROM users WHERE id = ?;"
)

func (a *Auth) CheckingTheUser(message *tgbotapi.Message) (*model2.User, error) {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(getUsersUserQuery, message.From.ID)
	if err != nil {
		return nil, errors.Wrap(err, "get user")
	}

	users, err := a.ReadUsers(rows)
	if err != nil {
		return nil, errors.Wrap(err, "read user")
	}

	switch len(users) {
	case 0:
		user := createSimpleUser(a.bot.LanguageInBot[0], message)
		if len(a.bot.LanguageInBot) > 1 && !administrator.ContainsInAdmin(message.From.ID) {
			user.Language = "not_defined"
		}
		referralID := a.pullReferralID(message)
		if err := a.addNewUser(user, a.bot.LanguageInBot[0], referralID); err != nil {
			return nil, errors.Wrap(err, "add new user")
		}

		model2.TotalIncome.WithLabelValues(
			a.bot.BotLink,
			a.bot.BotLang,
		).Inc()

		if user.Language == "not_defined" {
			return user, model2.ErrNotSelectedLanguage
		}
		return user, nil
	case 1:
		if users[0].Language == "not_defined" {
			return users[0], model2.ErrNotSelectedLanguage
		}
		return users[0], nil
	default:
		return nil, model2.ErrFoundTwoUsers
	}
}

func (a *Auth) SetStartLanguage(callback *tgbotapi.CallbackQuery) error {
	data := strings.Split(callback.Data, "?")[1]
	dataBase := a.bot.GetDataBase()
	_, err := dataBase.Exec("UPDATE users SET lang = ? WHERE id = ?", data, callback.From.ID)
	if err != nil {
		return err
	}
	return nil
}

func (a *Auth) addNewUser(user *model2.User, botLang string, referralID int64) error {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query("INSERT INTO users VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?);",
		user.ID,
		user.Balance,
		user.Completed,
		user.CompletedToday,
		user.LastVoice,
		user.AdvertChannel,
		user.ReferralCount,
		user.TakeBonus,
		user.Language,
		user.Status)
	if err != nil {
		return errors.Wrap(err, "query failed")
	}
	_ = rows.Close()

	if referralID == user.ID || referralID == 0 {
		return nil
	}

	baseUser, err := a.GetUser(referralID)
	if err != nil {
		return errors.Wrap(err, "get user")
	}

	// TODO: refactor accrual system

	baseUser.Balance += model2.AdminSettings.GetParams(botLang).ReferralAmount
	_, err = dataBase.Exec("UPDATE users SET balance = ?, referral_count = ? WHERE id = ?;",
		baseUser.Balance, baseUser.ReferralCount+1, baseUser.ID)
	if err != nil {
		text := "Fatal Err with DB - auth.85 //" + err.Error()
		a.msgs.SendNotificationToDeveloper(text, false)
		return err
	}

	return nil
}

func (a *Auth) pullReferralID(message *tgbotapi.Message) int64 {
	readParams := strings.Split(message.Text, " ")
	if len(readParams) < 2 {
		return 0
	}

	linkInfo, err := model2.DecodeLink(a.bot.GetDataBase(), readParams[1])
	if err != nil || linkInfo == nil {
		if err != nil {
			a.msgs.SendNotificationToDeveloper("some err in decode link: "+err.Error(), false)
		}

		model2.IncomeBySource.WithLabelValues(
			a.bot.BotLink,
			a.bot.BotLang,
			"unknown",
		).Inc()

		return 0
	}

	if err = a.saveIncomeUser(&model2.IncomeInfo{
		UserID: message.From.ID,
		Source: linkInfo.Source,
	}); err != nil {
		a.msgs.SendNotificationToDeveloper("some error in save income info: "+err.Error(), false)
	}

	model2.IncomeBySource.WithLabelValues(
		a.bot.BotLink,
		a.bot.BotLang,
		linkInfo.Source,
	).Inc()

	return linkInfo.ReferralID
}

func (a *Auth) saveIncomeUser(info *model2.IncomeInfo) error {
	_, err := a.bot.GetDataBase().Exec(`
INSERT INTO 
	income_info(user_id, source)
VALUES(?, ?);`,
		info.UserID,
		info.Source)
	if err != nil {
		return errors.Wrap(err, "failed insert income info")
	}

	return nil
}

func createSimpleUser(lang string, message *tgbotapi.Message) *model2.User {
	return &model2.User{
		ID:            message.From.ID,
		Language:      lang,
		AdvertChannel: rand.Intn(3) + 1,
		Status:        "active",
	}
}

func (a *Auth) GetUser(id int64) (*model2.User, error) {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(getUsersUserQuery, id)
	if err != nil {
		return nil, err
	}

	users, err := a.ReadUsers(rows)
	if err != nil || len(users) == 0 {
		return nil, model2.ErrUserNotFound
	}
	return users[0], nil
}

func (a *Auth) ReadUsers(rows *sql.Rows) ([]*model2.User, error) {
	defer rows.Close()

	var users []*model2.User

	for rows.Next() {
		user := &model2.User{}

		if err := rows.Scan(
			&user.ID,
			&user.Balance,
			&user.Completed,
			&user.CompletedToday,
			&user.LastVoice,
			&user.AdvertChannel,
			&user.ReferralCount,
			&user.TakeBonus,
			&user.Language,
			&user.Status); err != nil {
			a.msgs.SendNotificationToDeveloper(errors.Wrap(err, "failed to scan row").Error(), false)
		}

		users = append(users, user)
	}

	return users, nil
}
