package administrator

import (
	"strconv"
	"strings"

	"github.com/bots-empire/base-bot/msgs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/db"
	model2 "github.com/bots-empire/shazam-bot/internal/model"
)

const (
	bonusAmount         = "bonus_amount"
	minWithdrawalAmount = "min_withdrawal_amount"
	voiceAmount         = "voice_amount"
	voicePDAmount       = "voice_pd_amount"
	referralAmount      = "referral_amount"
	currencyType        = "currency_type"
)

func (a *Admin) MakeMoneySettingCommand(s *model2.Situation) error {
	markUp, text := a.sendMakeMoneyMenu(s.BotLang, s.User.ID)

	if db.RdbGetAdminMsgID(s.BotLang, s.User.ID) != 0 {
		err := a.msgs.NewEditMarkUpMessage(s.User.ID, db.RdbGetAdminMsgID(s.BotLang, s.User.ID), markUp, text)
		if err != nil {
			return errors.Wrap(err, "failed to edit markup")
		}
		err = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
		if err != nil {
			return errors.Wrap(err, "failed to send admin answer callback")
		}
		return nil
	}
	msgID, err := a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
	if err != nil {
		return errors.Wrap(err, "failed parse new id markup message")
	}
	db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)
	return nil
}

func (a *Admin) sendMakeMoneyMenu(botLang string, userID int64) (*tgbotapi.InlineKeyboardMarkup, string) {
	lang := model2.AdminLang(userID)
	text := a.bot.AdminText(lang, "make_money_setting_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("change_bonus_amount_button", "admin/make_money?bonus_amount")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_min_withdrawal_amount_button", "admin/make_money?min_withdrawal_amount")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_voice_amount_button", "admin/make_money?voice_amount")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_voice_pd_amount_button", "admin/make_money?voice_pd_amount")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_referral_amount_button", "admin/make_money?referral_amount")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_change_top_amount_button", "admin/change_top_amount_settings")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_currency_type_button", "admin/make_money?currency_type")),
		msgs.NewIlRow(msgs.NewIlAdminButton("add_task", "admin/add_task")),
		msgs.NewIlRow(msgs.NewIlAdminButton("get_all_tasks", "admin/get_all_tasks")),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_main_menu", "admin/send_menu")),
	).Build(a.bot.AdminLibrary[lang])

	db.RdbSetUser(botLang, userID, "admin/make_money_settings")
	return &markUp, text
}

func (a *Admin) ChangeParameterCommand(s *model2.Situation) error {
	changeParameter := strings.Split(s.CallbackQuery.Data, "?")[1]

	lang := model2.AdminLang(s.User.ID)
	var parameter, text string
	var value interface{}

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/make_money?"+changeParameter)

	switch changeParameter {
	case bonusAmount:
		parameter = a.bot.AdminText(lang, "change_bonus_amount_button")
		value = model2.AdminSettings.GetParams(s.BotLang).BonusAmount
	case minWithdrawalAmount:
		parameter = a.bot.AdminText(lang, "change_min_withdrawal_amount_button")
		value = model2.AdminSettings.GetParams(s.BotLang).MinWithdrawalAmount
	case voiceAmount:
		parameter = a.bot.AdminText(lang, "change_voice_amount_button")
		value = model2.AdminSettings.GetParams(s.BotLang).VoiceAmount
	case voicePDAmount:
		parameter = a.bot.AdminText(lang, "change_voice_pd_amount_button")
		value = model2.AdminSettings.GetParams(s.BotLang).MaxOfVoicePerDay
	case referralAmount:
		parameter = a.bot.AdminText(lang, "change_referral_amount_button")
		value = model2.AdminSettings.GetParams(s.BotLang).ReferralAmount
	case currencyType:
		parameter = a.bot.AdminText(lang, "change_currency_type_button")
		value = model2.AdminSettings.GetCurrency(s.BotLang)
	}

	text = a.adminFormatText(lang, "set_new_amount_text", parameter, value)
	//_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "type_the_text")
	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("back_to_make_money_setting")),
		msgs.NewRow(msgs.NewAdminButton("exit")),
	).Build(a.bot.AdminLibrary[lang])

	return a.msgs.NewParseMarkUpMessage(s.User.ID, markUp, text)
}

func (a *Admin) NotClickableButton(s *model2.Situation) error {
	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "not_clickable_button")
	return nil
}

func (a *Admin) SetTopAmountCommand(s *model2.Situation) error {
	lang := model2.AdminLang(s.User.ID)
	text := a.adminFormatText(lang, "change_top_settings_button")

	top := db.RdbGetTopLevelSetting(s.BotLang, s.User.ID)
	markUp := getTopSettingMenu(a.bot.AdminLibrary[lang], top+1, model2.AdminSettings.GlobalParameters[s.BotLang].Parameters.TopReward[top])

	msgID := db.RdbGetAdminMsgID(s.BotLang, s.User.ID)
	if msgID == 0 {
		id, err := a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
		if err != nil {
			return err
		}

		db.RdbSetAdminMsgID(s.BotLang, s.User.ID, id)
		return nil
	}

	return a.msgs.NewEditMarkUpMessage(s.User.ID, msgID, markUp, text)
}

func getTopSettingMenu(texts map[string]string, top int, amount int) *tgbotapi.InlineKeyboardMarkup {
	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("top_level_button", "admin/not_clickable")),
		msgs.NewIlRow(
			msgs.NewIlCustomButton("<<", "admin/change_top_level?dec"),
			msgs.NewIlCustomButton(strconv.Itoa(top), "admin/not_clickable"),
			msgs.NewIlCustomButton(">>", "admin/change_top_level?inc")),

		msgs.NewIlRow(msgs.NewIlAdminButton("top_amount_button", "admin/not_clickable")),
		msgs.NewIlRow(
			msgs.NewIlCustomButton("-5", "admin/change_top_amount?dec&5"),
			msgs.NewIlCustomButton("-1", "admin/change_top_amount?dec&1"),
			msgs.NewIlCustomButton(strconv.Itoa(amount), "admin/not_clickable"),
			msgs.NewIlCustomButton("+1", "admin/change_top_amount?inc&1"),
			msgs.NewIlCustomButton("+5", "admin/change_top_amount?inc&5")),

		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_make_money_setting", "admin/make_money_setting")),
	).Build(texts)

	return &markUp
}

func (a *Admin) ChangeTopLevelCommand(s *model2.Situation) error {
	level := db.RdbGetTopLevelSetting(s.BotLang, s.User.ID)
	operation := strings.Split(s.CallbackQuery.Data, "?")[1]

	switch operation {
	case "inc":
		if level == 2 {
			_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "already_max_level")
			return nil
		}
		level++
	case "dec":
		if level == 0 {
			_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "already_min_level")
			return nil
		}
		level--
	}

	db.RdbSetTopLevelSetting(s.BotLang, s.User.ID, level)
	return a.SetTopAmountCommand(s)
}

func (a *Admin) ChangeTopAmountButtonCommand(s *model2.Situation) error {
	level := db.RdbGetTopLevelSetting(s.BotLang, s.User.ID)

	allParams := strings.Split(s.CallbackQuery.Data, "?")[1]
	changeParams := strings.Split(allParams, "&")
	operation := changeParams[0]

	switch operation {
	case "inc":
		value, _ := strconv.Atoi(changeParams[1])
		model2.AdminSettings.GetParams(s.BotLang).TopReward[level] += value
	case "dec":
		value, _ := strconv.Atoi(changeParams[1])

		if model2.AdminSettings.GetParams(s.BotLang).TopReward[level]-value < 1 {
			_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "already_min_value")
			return nil
		}

		model2.AdminSettings.GetParams(s.BotLang).TopReward[level] -= value
	}

	model2.SaveAdminSettings()
	return a.SetTopAmountCommand(s)
}
