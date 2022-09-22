package administrator

import (
	"strconv"
	"strings"

	"github.com/bots-empire/base-bot/msgs"

	"github.com/bots-empire/shazam-bot/internal/db"
	model2 "github.com/bots-empire/shazam-bot/internal/model"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (a *Admin) StartMailingCommand(s *model2.Situation) error {
	channel, _ := strconv.Atoi(strings.Split(s.CallbackQuery.Data, "?")[1])

	err := a.mailing.StartMailing(channelsFromNum(channel))
	if err != nil {
		return err
	}

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "mailing_successful")
	if channel == model2.GlobalMailing {
		return a.AdvertisementMenuCommand(s)
	}
	return a.resendAdvertisementMenuLevel(s.BotLang, s.User.ID, channel)
}

func channelsFromNum(channel int) []int {
	if channel == 4 {
		return []int{1, 2, 3}
	}

	return []int{channel}
}

func (a *Admin) SelectedLangCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")
	partition := data[1]
	lang := data[2]
	switch partition {
	case "switch_lang":
		switchLangOnKeyboard(lang)
		if err := a.sendMailingMenu(s.BotLang, s.User.ID, "1"); err != nil {
			return err
		}
		return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	case "switch_all":
		a.switchedSelectedLanguages()
		if err := a.sendMailingMenu(s.BotLang, s.User.ID, "1"); err != nil {
			return err
		}

		return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	}
	return nil
}

func (a *Admin) sendMailingMenu(botLang string, userID int64, channel string) error {
	lang := model2.AdminLang(userID)

	text := a.bot.AdminText(lang, "mailing_main_text")
	markUp := createMailingMarkUp(botLang, channel, a.bot.AdminLibrary[lang])

	if db.RdbGetAdminMsgID(botLang, userID) == 0 {
		msgID, err := a.msgs.NewIDParseMarkUpMessage(userID, &markUp, text)
		if err != nil {
			return err
		}

		db.RdbSetAdminMsgID(botLang, userID, msgID)
		return nil
	}

	return a.msgs.NewEditMarkUpMessage(userID, db.RdbGetAdminMsgID(botLang, userID), &markUp, text)
}

func createMailingMarkUp(botLang, channel string, texts map[string]string) tgbotapi.InlineKeyboardMarkup {
	markUp := &msgs.InlineMarkUp{}

	if buttonUnderAdvertisementUnable(botLang) {
		markUp.Rows = append(markUp.Rows,
			msgs.NewIlRow(msgs.NewIlAdminButton("advert_button_on", "admin/change_advert_button_status?"+channel)),
		)
	} else {
		markUp.Rows = append(markUp.Rows,
			msgs.NewIlRow(msgs.NewIlAdminButton("advert_button_off", "admin/change_advert_button_status?"+channel)),
		)
	}

	if channel == "4" {
		markUp.Rows = append(markUp.Rows,
			msgs.NewIlRow(msgs.NewIlAdminButton("start_mailing_button", "admin/start_mailing?"+channel)),
			msgs.NewIlRow(msgs.NewIlAdminButton("back_to_chan_menu", "admin/advertisement")),
		)
	} else {
		markUp.Rows = append(markUp.Rows,
			msgs.NewIlRow(msgs.NewIlAdminButton("start_mailing_button", "admin/start_mailing?"+channel)),
			msgs.NewIlRow(msgs.NewIlAdminButton("back_to_advertisement_setting", "admin/change_advert_chan?"+channel)),
		)
	}

	return markUp.Build(texts)
}

func switchLangOnKeyboard(lang string) {
	model2.AdminSettings.GlobalParameters[lang].LangSelectedMap[lang] = !model2.AdminSettings.GlobalParameters[lang].LangSelectedMap[lang]
	model2.SaveAdminSettings()
}

func (a *Admin) resendAdvertisementMenuLevel(botLang string, userID int64, channel int) error {
	db.DeleteOldAdminMsg(botLang, userID)

	db.RdbSetUser(botLang, userID, "admin/advertisement")
	inlineMarkUp, text := a.getAdvertisementMenu(botLang, userID, channel)
	msgID, err := a.msgs.NewIDParseMarkUpMessage(userID, inlineMarkUp, text)
	if err != nil {
		return err
	}
	db.RdbSetAdminMsgID(botLang, userID, msgID)
	return nil
}

func (a *Admin) switchedSelectedLanguages() {
	if a.selectedAllLanguage() {
		resetSelectedLang()
		return
	}
	chooseAllLanguages()
}

func resetSelectedLang() {
	for lang := range model2.AdminSettings.GlobalParameters {
		model2.AdminSettings.GlobalParameters[lang].LangSelectedMap[lang] = false
	}
	model2.SaveAdminSettings()
}

func chooseAllLanguages() {
	for lang := range model2.AdminSettings.GlobalParameters {
		model2.AdminSettings.GlobalParameters[lang].LangSelectedMap[lang] = true
	}
	model2.SaveAdminSettings()
}

func (a *Admin) selectedAllLanguage() bool {
	for _, lang := range a.bot.LanguageInBot {
		if !model2.AdminSettings.GlobalParameters[lang].LangSelectedMap[lang] {
			return false
		}
	}
	return true
}

func buttonUnderAdvertisementUnable(botLang string) bool {
	return model2.AdminSettings.GlobalParameters[botLang].Parameters.ButtonUnderAdvert
}
