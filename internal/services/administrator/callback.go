package administrator

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bots-empire/base-bot/msgs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/db"
	model2 "github.com/bots-empire/shazam-bot/internal/model"
)

type AdminCallbackHandlers struct {
	Handlers map[string]model2.Handler
}

func (h *AdminCallbackHandlers) GetHandler(command string) model2.Handler {
	return h.Handlers[command]
}

func (h *AdminCallbackHandlers) Init(adminSrv *Admin) {
	//Admin Setting command
	h.OnCommand("/send_menu", adminSrv.AdminMenuCommand)
	h.OnCommand("/admin_setting", adminSrv.AdminSettingCommand)
	h.OnCommand("/change_language", adminSrv.ChangeLangCommand)
	h.OnCommand("/set_language", adminSrv.SetNewLangCommand)
	h.OnCommand("/send_admin_list", adminSrv.AdminListCommand)
	h.OnCommand("/add_admin_msg", adminSrv.NewAdminToListCommand)
	h.OnCommand("/delete_admin", adminSrv.DeleteAdminCommand)
	h.OnCommand("/send_advert_source_menu", adminSrv.AdvertSourceMenuCommand)
	h.OnCommand("/add_new_source", adminSrv.AddNewSourceCommand)

	//Make Money Setting command
	h.OnCommand("/make_money_setting", adminSrv.MakeMoneySettingCommand)
	h.OnCommand("/make_money", adminSrv.ChangeParameterCommand)

	//Mailing command
	h.OnCommand("/advertisement", adminSrv.AdvertisementMenuCommand)
	h.OnCommand("/change_advert_chan", adminSrv.AdvertisementChanMenuCommand)
	h.OnCommand("/change_url_menu", adminSrv.ChangeUrlMenuCommand)
	h.OnCommand("/change_top_amount_settings", adminSrv.SetTopAmountCommand)
	h.OnCommand("/change_top_level", adminSrv.ChangeTopLevelCommand)
	h.OnCommand("/change_top_amount", adminSrv.ChangeTopAmountButtonCommand)
	h.OnCommand("/not_clickable", adminSrv.NotClickableButton)
	h.OnCommand("/change_text_menu", adminSrv.ChangeTextMenuCommand)
	h.OnCommand("/change_photo_menu", adminSrv.ChangePhotoMenuCommand)
	h.OnCommand("/change_video_menu", adminSrv.ChangeVideoMenuCommand)
	h.OnCommand("/turn", adminSrv.TurnMenuCommand)
	h.OnCommand("/change_advert_button_status", adminSrv.ChangeUnderAdvertButtonCommand)
	h.OnCommand("/mailing_menu", adminSrv.MailingMenuCommand)
	h.OnCommand("/send_advertisement", adminSrv.SelectedLangCommand)
	h.OnCommand("/start_mailing", adminSrv.StartMailingCommand)

	//Send Statistic command
	h.OnCommand("/send_statistic", adminSrv.StatisticCommand)
}

func (h *AdminCallbackHandlers) OnCommand(command string, handler model2.Handler) {
	h.Handlers[command] = handler
}

func (a *Admin) CheckAdminCallback(s *model2.Situation) error {
	if !ContainsInAdmin(s.User.ID) {
		return a.notAdmin(s.User)
	}

	s.Command = strings.TrimLeft(s.Command, "admin")

	Handler := model2.Bots[s.BotLang].AdminCallBackHandler.GetHandler(s.Command)
	if Handler != nil {
		return Handler(s)
	}
	return model2.ErrCommandNotConverted
}

func (a *Admin) AdminLoginCommand(s *model2.Situation) error {
	if !ContainsInAdmin(s.User.ID) {
		return a.notAdmin(s.User)
	}

	updateFirstNameInfo(s.Message)
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	if err := a.setAdminBackButton(s.User.ID, "admin_log_in"); err != nil {
		return err
	}
	s.Command = "/send_menu"
	return a.AdminMenuCommand(s)
}

func ContainsInAdmin(userID int64) bool {
	for key := range model2.AdminSettings.AdminID {
		if key == userID {
			return true
		}
	}
	return false
}

func (a *Admin) notAdmin(user *model2.User) error {
	text := a.bot.LangText(user.Language, "not_admin")
	return a.msgs.SendSimpleMsg(user.ID, text)
}

func updateFirstNameInfo(message *tgbotapi.Message) {
	userID := message.From.ID
	model2.AdminSettings.AdminID[userID].FirstName = message.From.FirstName
	model2.SaveAdminSettings()
}

func (a *Admin) setAdminBackButton(userID int64, key string) error {
	lang := model2.AdminLang(userID)
	text := a.bot.AdminText(lang, key)

	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("admin_log_out_text")),
	).Build(a.bot.AdminLibrary[lang])

	return a.msgs.NewParseMarkUpMessage(userID, markUp, text)
}

func (a *Admin) AdminMenuCommand(s *model2.Situation) error {
	if strings.Contains(s.Params.Level, "make_money?") && s.Message.Text == "← Назад к ⚙️ Заработок" {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	}

	db.RdbSetUser(s.BotLang, s.User.ID, "admin")
	lang := model2.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "admin_main_menu_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("setting_admin_button", "admin/admin_setting")),
		msgs.NewIlRow(msgs.NewIlAdminButton("setting_make_money_button", "admin/make_money_setting")),
		msgs.NewIlRow(msgs.NewIlAdminButton("setting_advertisement_button", "admin/advertisement")),
		msgs.NewIlRow(msgs.NewIlAdminButton("setting_statistic_button", "admin/send_statistic")),
	).Build(a.bot.AdminLibrary[lang])

	if db.RdbGetAdminMsgID(s.BotLang, s.User.ID) != 0 {
		_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
		return a.msgs.NewEditMarkUpMessage(
			s.User.ID,
			db.RdbGetAdminMsgID(s.BotLang, s.User.ID),
			&markUp,
			text,
		)
	}
	msgID, err := a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
	if err != nil {
		return err
	}
	db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)
	return nil
}

func (a *Admin) AdminSettingCommand(s *model2.Situation) error {
	if strings.Contains(s.Params.Level, "delete_admin") {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	}

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/mailing")
	lang := model2.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "admin_setting_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("setting_language_button", "admin/change_language")),
		msgs.NewIlRow(msgs.NewIlAdminButton("admin_list_button", "admin/send_admin_list")),
		msgs.NewIlRow(msgs.NewIlAdminButton("advertisement_source_button", "admin/send_advert_source_menu")),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_main_menu", "admin/send_menu")),
	).Build(a.bot.AdminLibrary[lang])
	if err := a.sendMsgAdnAnswerCallback(s, &markUp, text); err != nil {
		return err
	}
	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
}

func (a *Admin) ChangeLangCommand(s *model2.Situation) error {
	lang := model2.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "admin_set_lang_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("set_lang_en", "admin/set_language?en"),
			msgs.NewIlAdminButton("set_lang_ru", "admin/set_language?ru")),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_admin_settings", "admin/admin_setting")),
	).Build(a.bot.AdminLibrary[lang])

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	return a.msgs.NewEditMarkUpMessage(s.User.ID, db.RdbGetAdminMsgID(s.BotLang, s.User.ID), &markUp, text)
}

func (a *Admin) SetNewLangCommand(s *model2.Situation) error {
	lang := strings.Split(s.CallbackQuery.Data, "?")[1]
	model2.AdminSettings.AdminID[s.User.ID].Language = lang
	model2.SaveAdminSettings()

	if err := a.setAdminBackButton(s.User.ID, "language_set"); err != nil {
		return err
	}
	s.Command = "admin/admin_setting"
	return a.AdminSettingCommand(s)
}

func (a *Admin) AdvertisementMenuCommand(s *model2.Situation) error {
	if strings.Contains(s.Params.Level, "change_text_url?") {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	}

	lang := model2.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "change_advert_chan_text")

	msgID := db.RdbGetAdminMsgID(s.BotLang, s.User.ID)
	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("change_advert_chan_1", "admin/change_advert_chan?1")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_advert_chan_2", "admin/change_advert_chan?2")),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_advert_chan_3", "admin/change_advert_chan?3")),
		msgs.NewIlRow(msgs.NewIlAdminButton("global_advertisement", "admin/change_advert_chan?"+strconv.Itoa(model2.MainAdvert))),
		msgs.NewIlRow(msgs.NewIlAdminButton("distribute_button_general", "admin/mailing_menu?"+strconv.Itoa(model2.GlobalMailing))),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_main_menu", "admin/send_menu")),
	).Build(a.bot.AdminLibrary[lang])

	if msgID == 0 {
		var err error
		msgID, err = a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
		if err != nil {
			return err
		}

		db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)
	} else {
		if err := a.msgs.NewEditMarkUpMessage(s.User.ID, msgID, &markUp, text); err != nil {
			return err
		}
	}

	if s.CallbackQuery != nil {
		if s.CallbackQuery.ID != "" {
			if err := a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Admin) AdvertisementChanMenuCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")

	channel, _ := strconv.Atoi(data[1])

	if channel == 5 {
		markUp, text := a.getAdvertUrlMenu(s.BotLang, s.User.ID, channel)
		msgID := db.RdbGetAdminMsgID(s.BotLang, s.User.ID)
		if msgID == 0 {
			var err error
			msgID, err = a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
			if err != nil {
				return err
			}
			db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)
			return nil
		} else {
			return a.msgs.NewEditMarkUpMessage(s.User.ID, msgID, markUp, text)
		}
	}

	if strings.Contains(s.Params.Level, "change_text_url?") {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	}

	markUp, text := a.getAdvertisementMenu(s.BotLang, s.User.ID, channel)
	msgID := db.RdbGetAdminMsgID(s.BotLang, s.User.ID)
	if msgID == 0 {
		var err error
		msgID, err = a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
		if err != nil {
			return err
		}

		db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)
	} else {
		if err := a.msgs.NewEditMarkUpMessage(s.User.ID, msgID, markUp, text); err != nil {
			return err
		}
	}

	if s.CallbackQuery != nil {
		if s.CallbackQuery.ID != "" {
			if err := a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice"); err != nil {
				return err
			}
		}
	}
	return nil
}

func (a *Admin) getAdvertUrlMenu(botLang string, userID int64, channel int) (*tgbotapi.InlineKeyboardMarkup, string) {
	lang := model2.AdminLang(userID)
	text := a.adminFormatText(lang, "advertisement_setting_text", "Главный")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("change_url_button", "admin/change_url_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_chan_menu", "admin/advertisement")),
	).Build(a.bot.AdminLibrary[lang])

	db.RdbSetUser(botLang, userID, "admin/change_advert_chan_"+strconv.Itoa(channel))
	return &markUp, text
}

func (a *Admin) getAdvertisementMenu(botLang string, userID int64, channel int) (*tgbotapi.InlineKeyboardMarkup, string) {
	lang := model2.AdminLang(userID)
	text := a.adminFormatText(lang, "advertisement_setting_text", strconv.Itoa(channel))

	Photo := "photo"
	Video := "video"
	Nothing := "nothing"

	switch model2.AdminSettings.GlobalParameters[botLang].AdvertisingChoice[channel] {
	case "photo":
		Photo = "photo_on"
	case "video":
		Video = "video_on"
	default:
		Nothing = "nothing_on"
	}

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("change_url_button", "admin/change_url_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_text_button", "admin/change_text_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_photo_button", "admin/change_photo_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(msgs.NewIlAdminButton("change_video_button", "admin/change_video_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(
			msgs.NewIlAdminButton("turn_"+Photo, "admin/turn?photo?"+strconv.Itoa(channel)),
			msgs.NewIlAdminButton("turn_"+Video, "admin/turn?video?"+strconv.Itoa(channel)),
			msgs.NewIlAdminButton("turn_"+Nothing, "admin/turn?nothing?"+strconv.Itoa(channel)),
		),
		msgs.NewIlRow(msgs.NewIlAdminButton("distribute_button", "admin/mailing_menu?"+strconv.Itoa(channel))),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_chan_menu", "admin/advertisement")),
	).Build(a.bot.AdminLibrary[lang])

	db.RdbSetUser(botLang, userID, "admin/change_advert_chan_"+strconv.Itoa(channel))
	return &markUp, text
}

func (a *Admin) ChangeUrlMenuCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")
	channel, _ := strconv.Atoi(data[1])

	key := "set_new_url_text"
	value := model2.AdminSettings.GetAdvertUrl(s.BotLang, channel)

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/change_text_url?change_url?"+data[1])
	if err := a.promptForInput(s.User.ID, key, value); err != nil {
		return err
	}
	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "type_the_text")
}

func (a *Admin) ChangeTextMenuCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")
	channel, _ := strconv.Atoi(data[1])

	key := "set_new_advertisement_text"
	value := model2.AdminSettings.GetAdvertText(s.BotLang, channel)

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/change_text_url?change_text?"+data[1])
	if err := a.promptForInput(s.User.ID, key, value); err != nil {
		return err
	}
	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "type_the_text")
}

func (a *Admin) ChangePhotoMenuCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")
	channel, _ := strconv.Atoi(data[1])

	lang := model2.AdminLang(s.User.ID)
	key := "set_new_advertisement_photo"

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/change_text_url?change_photo?"+data[1])
	err := a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "send_photo")
	if err != nil {
		return err
	}

	text := a.bot.AdminText(model2.AdminLang(s.User.ID), key)

	photoFileBytes := tgbotapi.FileID(model2.AdminSettings.GlobalParameters[s.BotLang].AdvertisingPhoto[channel])

	if photoFileBytes == "" {
		key = "no_photo_found"
		text = a.bot.AdminText(model2.AdminLang(s.User.ID), key)
		markUp := msgs.NewMarkUp(
			msgs.NewRow(msgs.NewAdminButton("back_to_advertisement_setting")),
			msgs.NewRow(msgs.NewAdminButton("exit")),
		).Build(a.bot.AdminLibrary[lang])
		return a.msgs.NewParseMarkUpMessage(s.User.ID, &markUp, text)
	}

	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("back_to_advertisement_setting")),
		msgs.NewRow(msgs.NewAdminButton("exit")),
	).Build(a.bot.AdminLibrary[lang])

	return a.msgs.NewParseMarkUpPhotoMessage(s.User.ID, &markUp, text, photoFileBytes)
}

func (a *Admin) ChangeVideoMenuCommand(s *model2.Situation) error {
	data := strings.Split(s.CallbackQuery.Data, "?")
	channel, _ := strconv.Atoi(data[1])

	lang := model2.AdminLang(s.User.ID)
	key := "set_new_advertisement_video"

	db.RdbSetUser(s.BotLang, s.User.ID, "admin/change_text_url?change_video?"+data[1])
	err := a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "send_the_video")
	if err != nil {
		return err
	}

	text := a.bot.AdminText(model2.AdminLang(s.User.ID), key)

	videoFileBytes := tgbotapi.FileID(model2.AdminSettings.GlobalParameters[s.BotLang].AdvertisingVideo[channel])

	if videoFileBytes == "" {
		key = "no_video_found"
		text = a.bot.AdminText(model2.AdminLang(s.User.ID), key)
		markUp := msgs.NewMarkUp(
			msgs.NewRow(msgs.NewAdminButton("back_to_advertisement_setting")),
			msgs.NewRow(msgs.NewAdminButton("exit")),
		).Build(a.bot.AdminLibrary[lang])
		return a.msgs.NewParseMarkUpMessage(s.User.ID, &markUp, text)
	}

	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("back_to_advertisement_setting")),
		msgs.NewRow(msgs.NewAdminButton("exit")),
	).Build(a.bot.AdminLibrary[lang])

	return a.msgs.NewParseMarkUpVideoMessage(s.User.ID, &markUp, text, videoFileBytes)
}

func (a *Admin) TurnMenuCommand(s *model2.Situation) error {

	//	lang := assets.AdminLang(s.User.ID)
	data := strings.Split(s.CallbackQuery.Data, "?")
	channel, _ := strconv.Atoi(data[2])
	switch data[1] {
	case "photo":
		if model2.AdminSettings.GlobalParameters[s.BotLang].AdvertisingPhoto[channel] == "" {
			return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "add_media")
		}
	case "video":
		if model2.AdminSettings.GlobalParameters[s.BotLang].AdvertisingVideo[channel] == "" {
			return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "add_media")
		}
	}
	model2.AdminSettings.UpdateAdvertChoice(s.BotLang, channel, data[1])

	err := a.msgs.SendAdminAnswerCallback(s.CallbackQuery, data[1])
	if err != nil {
		return err
	}
	//db.DeleteOldAdminMsg(lang, s.User.ID)

	callback := &tgbotapi.CallbackQuery{
		Data: "admin/change_advert_chan?" + data[2],
	}

	s.CallbackQuery = callback
	return a.AdvertisementChanMenuCommand(s)
}

func (a *Admin) ChangeUnderAdvertButtonCommand(s *model2.Situation) error {
	channel := strings.Split(s.CallbackQuery.Data, "?")[1]

	model2.AdminSettings.GlobalParameters[s.BotLang].Parameters.ButtonUnderAdvert =
		!model2.AdminSettings.GlobalParameters[s.BotLang].Parameters.ButtonUnderAdvert
	model2.SaveAdminSettings()

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	return a.sendMailingMenu(s.BotLang, s.CallbackQuery.From.ID, channel)
}

func (a *Admin) MailingMenuCommand(s *model2.Situation) error {
	channel := strings.Split(s.CallbackQuery.Data, "?")[1]
	db.RdbSetUser(s.BotLang, s.User.ID, "admin/mailing")
	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	return a.sendMailingMenu(s.BotLang, s.User.ID, channel)
}

func (a *Admin) promptForInput(userID int64, key string, values ...interface{}) error {
	lang := model2.AdminLang(userID)

	text := a.adminFormatText(lang, key, values...)
	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("back_to_advertisement_setting")),
		msgs.NewRow(msgs.NewAdminButton("exit")),
	).Build(a.bot.AdminLibrary[lang])

	return a.msgs.NewParseMarkUpMessage(userID, markUp, text)
}

func (a *Admin) StatisticCommand(s *model2.Situation) error {
	lang := model2.AdminLang(s.User.ID)

	count := a.CountUsers()
	allCount := a.countAllUsers()
	referrals := a.countReferrals(s.BotLang, count)
	blocked := a.countBlockedUsers(s.BotLang)
	subscribers := a.countSubscribers(s.BotLang)
	text := a.adminFormatText(lang, "statistic_text",
		allCount, count, referrals, blocked, subscribers, count-blocked)

	if err := a.msgs.NewParseMessage(s.User.ID, text); err != nil {
		return err
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "/send_menu"
	if err := a.AdminMenuCommand(s); err != nil {
		return err
	}

	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
}

func (a *Admin) adminFormatText(lang, key string, values ...interface{}) string {
	formatText := a.bot.AdminText(lang, key)
	return fmt.Sprintf(formatText, values...)
}

func (a *Admin) sendMsgAdnAnswerCallback(s *model2.Situation, markUp *tgbotapi.InlineKeyboardMarkup, text string) error {
	if db.RdbGetAdminMsgID(s.BotLang, s.User.ID) != 0 {
		return a.msgs.NewEditMarkUpMessage(s.User.ID, db.RdbGetAdminMsgID(s.BotLang, s.User.ID), markUp, text)
	}
	msgID, err := a.msgs.NewIDParseMarkUpMessage(s.User.ID, markUp, text)
	if err != nil {
		return err
	}
	db.RdbSetAdminMsgID(s.BotLang, s.User.ID, msgID)

	if s.CallbackQuery != nil {
		if s.CallbackQuery.ID != "" {
			return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
		}
	}
	return nil
}
