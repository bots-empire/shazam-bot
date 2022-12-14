package auth

import (
	"database/sql"
	"math/rand"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/model"
	"github.com/bots-empire/shazam-bot/internal/services/administrator"
)

const (
	getUsersUserQuery = `SELECT id, balance, completed, completed_today, last_shazam, father_id, all_referrals, advert_channel, referral_count, take_bonus, lang, status FROM shazam.users WHERE id = $1;`
)

func (a *Auth) CheckingTheUser(message *tgbotapi.Message) (*model.User, error) {
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

		model.TotalIncome.WithLabelValues(
			a.bot.BotLink,
			a.bot.BotLang,
		).Inc()

		if user.Language == "not_defined" {
			return user, model.ErrNotSelectedLanguage
		}
		return user, nil
	case 1:
		if users[0].Language == "not_defined" {
			return users[0], model.ErrNotSelectedLanguage
		}
		return users[0], nil
	default:
		return nil, model.ErrFoundTwoUsers
	}
}

func (a *Auth) SetStartLanguage(callback *tgbotapi.CallbackQuery) error {
	data := strings.Split(callback.Data, "?")[1]
	dataBase := a.bot.GetDataBase()
	_, err := dataBase.Exec("UPDATE shazam.users SET lang = $1 WHERE id = $2", data, callback.From.ID)
	if err != nil {
		return err
	}
	return nil
}

func (a *Auth) addNewUser(user *model.User, botLang string, referralID int64) error {
	if referralID == user.ID {
		referralID = 0
	}

	rows, err := a.bot.GetDataBase().Query(`
INSERT INTO shazam.users (id,
    balance,
    completed,
    completed_today,
    last_shazam,
    father_id,
    all_referrals,
    advert_channel,
    referral_count,
    take_bonus,
    lang,
    status) VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12);`,
		user.ID,
		user.Balance,
		user.Completed,
		user.CompletedToday,
		user.LastShazam,
		user.FatherID,
		user.AllReferrals,
		user.AdvertChannel,
		user.ReferralCount,
		user.TakeBonus,
		user.Language,
		user.Status)
	if err != nil {
		return errors.Wrap(err, "query failed")
	}
	_ = rows.Close()

	if referralID == 0 {
		return nil
	}
	_, err = a.bot.GetDataBase().Exec(`
UPDATE shazam.users SET
	referral_count = referral_count + 1
WHERE id = $1;`,
		referralID)
	if err != nil {
		return err
	}

	return a.referralRewardSystem(botLang, referralID, 1)
}

func (a *Auth) pullReferralID(message *tgbotapi.Message) int64 {
	readParams := strings.Split(message.Text, " ")
	if len(readParams) < 2 {
		return 0
	}

	linkInfo, err := model.DecodeLink(a.bot.GetDataBase(), readParams[1])
	if err != nil || linkInfo == nil {
		if err != nil {
			a.msgs.SendNotificationToDeveloper("some err in decode link: "+err.Error(), false)
		}

		model.IncomeBySource.WithLabelValues(
			a.bot.BotLink,
			a.bot.BotLang,
			"unknown",
		).Inc()

		return 0
	}

	if err = a.saveIncomeUser(&model.IncomeInfo{
		UserID: message.From.ID,
		Source: linkInfo.Source,
	}); err != nil {
		a.msgs.SendNotificationToDeveloper("some error in save income info: "+err.Error(), false)
	}

	model.IncomeBySource.WithLabelValues(
		a.bot.BotLink,
		a.bot.BotLang,
		linkInfo.Source,
	).Inc()

	return linkInfo.ReferralID
}

func (a *Auth) saveIncomeUser(info *model.IncomeInfo) error {
	_, err := a.bot.GetDataBase().Exec(`
INSERT INTO 
	shazam.income_info(user_id, source)
VALUES($1, $2);`,
		info.UserID,
		info.Source)
	if err != nil {
		return errors.Wrap(err, "failed insert income info")
	}

	return nil
}

func createSimpleUser(lang string, message *tgbotapi.Message) *model.User {
	return &model.User{
		ID:            message.From.ID,
		Language:      lang,
		AdvertChannel: rand.Intn(3) + 1,
		Status:        "active",
	}
}

func (a *Auth) GetUser(id int64) (*model.User, error) {
	dataBase := a.bot.GetDataBase()
	rows, err := dataBase.Query(getUsersUserQuery, id)
	if err != nil {
		return nil, err
	}

	users, err := a.ReadUsers(rows)
	if err != nil || len(users) == 0 {
		return nil, model.ErrUserNotFound
	}
	return users[0], nil
}

func (a *Auth) ReadUsers(rows *sql.Rows) ([]*model.User, error) {
	defer rows.Close()

	var users []*model.User

	for rows.Next() {
		user := &model.User{}

		if err := rows.Scan(
			&user.ID,
			&user.Balance,
			&user.Completed,
			&user.CompletedToday,
			&user.LastShazam,
			&user.FatherID,
			&user.AllReferrals,
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
