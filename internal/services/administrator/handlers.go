package administrator

import (
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/db"
	model2 "github.com/bots-empire/shazam-bot/internal/model"
)

type AdminMessagesHandlers struct {
	Handlers map[string]model2.Handler
}

func (h *AdminMessagesHandlers) GetHandler(command string) model2.Handler {
	return h.Handlers[command]
}

func (h *AdminMessagesHandlers) Init(adminSrv *Admin) {
	//Delete Admin command
	h.OnCommand("/delete_admin", adminSrv.RemoveAdminCommand)

	//Change Advertisement parameters command
	h.OnCommand("/make_money", adminSrv.UpdateParameterCommand)
	h.OnCommand("/change_text_url", adminSrv.SetNewTextUrlCommand)
	h.OnCommand("/advertisement_setting", adminSrv.AdvertisementSettingCommand)
	h.OnCommand("/get_new_source", adminSrv.GetNewSourceCommand)
}

func (h *AdminMessagesHandlers) OnCommand(command string, handler model2.Handler) {
	h.Handlers[command] = handler
}

func (a *Admin) CheckAdminMessage(s *model2.Situation) error {
	if !ContainsInAdmin(s.User.ID) {
		return a.notAdmin(s.User)
	}

	s.Command, s.Err = a.bot.GetCommandFromText(s.Message, s.User.Language, s.User.ID)
	if s.Err == nil {
		Handler := model2.Bots[s.BotLang].AdminMessageHandler.
			GetHandler(s.Command)

		if Handler != nil {
			return Handler(s)
		}
	}

	s.Command = strings.TrimLeft(strings.Split(s.Params.Level, "?")[0], "admin")

	Handler := model2.Bots[s.BotLang].AdminMessageHandler.
		GetHandler(s.Command)

	if Handler != nil {
		return Handler(s)
	}

	if a.checkIncomeInfo(s) {
		return nil
	}

	return model2.ErrCommandNotConverted
}

func (a *Admin) checkIncomeInfo(s *model2.Situation) bool {
	if s.Message == nil {
		return false
	}

	if s.Message.ForwardFrom == nil {
		return false
	}

	lang := model2.AdminLang(s.User.ID)

	info, err := a.getIncomeInfo(s.Message.ForwardFrom.ID)
	if err != nil {
		a.msgs.SendNotificationToDeveloper("some error in get income info: "+err.Error(), false)
		return true
	}

	if info == nil {
		err = a.msgs.NewParseMessage(s.User.ID, a.bot.AdminText(lang, "user_info_not_found"))
		return true
	}

	err = a.msgs.NewParseMessage(s.User.ID, a.adminFormatText(lang, "user_income_info", info.UserID, info.Source))
	if err != nil {
		a.msgs.SendNotificationToDeveloper("error in send msg: "+err.Error(), false)
		return true
	}

	return true
}

func (a *Admin) RemoveAdminCommand(s *model2.Situation) error {
	lang := model2.AdminLang(s.User.ID)
	adminId, err := strconv.ParseInt(s.Message.Text, 10, 64)
	if err != nil {
		text := a.bot.AdminText(lang, "incorrect_admin_id_text")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	if !checkAdminIDInTheList(adminId) {
		text := a.bot.AdminText(lang, "incorrect_admin_id_text")
		return a.msgs.NewParseMessage(s.User.ID, text)

	}

	delete(model2.AdminSettings.AdminID, adminId)
	model2.SaveAdminSettings()
	if err := a.setAdminBackButton(s.User.ID, "admin_removed_status"); err != nil {
		return err
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	s.Command = "admin/send_admin_list"
	s.CallbackQuery = &tgbotapi.CallbackQuery{Data: "admin/send_admin_list"}
	return a.AdminListCommand(s)
}

func checkAdminIDInTheList(adminID int64) bool {
	_, inMap := model2.AdminSettings.AdminID[adminID]
	return inMap
}

func (a *Admin) UpdateParameterCommand(s *model2.Situation) error {
	if strings.Contains(s.Params.Level, "make_money?") && s.Message.Text == "← Назад к ⚙️ Заработок" {
		if err := a.setAdminBackButton(s.User.ID, "operation_canceled"); err != nil {
			return err
		}
		db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
		s.Command = "admin/make_money_setting"

		return a.MakeMoneySettingCommand(s)
	}

	partition := strings.Split(s.Params.Level, "?")[1]

	if partition == currencyType {
		model2.AdminSettings.UpdateCurrency(s.BotLang, s.Message.Text)
	} else {
		err := a.setNewIntParameter(s, partition)
		if err != nil {
			return err
		}
	}

	model2.SaveAdminSettings()
	err := a.setAdminBackButton(s.User.ID, "operation_completed")
	if err != nil {
		return nil
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "admin/make_money_setting"

	return a.MakeMoneySettingCommand(s)
}

func (a *Admin) setNewIntParameter(s *model2.Situation, partition string) error {
	lang := model2.AdminLang(s.User.ID)

	newAmount, err := strconv.Atoi(s.Message.Text)
	if err != nil || newAmount <= 0 {
		text := a.bot.AdminText(lang, "incorrect_make_money_change_input")
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	switch partition {
	case bonusAmount:
		model2.AdminSettings.UpdateBonusAmount(s.BotLang, newAmount)
	case minWithdrawalAmount:
		model2.AdminSettings.UpdateMinWithdrawalAmount(s.BotLang, newAmount)
	case voiceAmount:
		model2.AdminSettings.UpdateVoiceAmount(s.BotLang, newAmount)
	case voicePDAmount:
		model2.AdminSettings.UpdateMaxOfVoicePerDay(s.BotLang, newAmount)
	case referralAmount:
		model2.AdminSettings.UpdateReferralAmount(s.BotLang, newAmount)
	}

	return nil
}

func (a *Admin) SetNewTextUrlCommand(s *model2.Situation) error {
	capitation := strings.Split(s.Params.Level, "?")[1]
	channel, _ := strconv.Atoi(strings.Split(s.Params.Level, "?")[2])
	lang := model2.AdminLang(s.User.ID)
	status := "operation_canceled"

	switch capitation {
	case "change_url":
		url, chatID := getUrlAndChatID(s.Message)
		if chatID == 0 {
			text := a.bot.AdminText(lang, "chat_id_not_update")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model2.AdminSettings.UpdateAdvertChannelID(s.BotLang, chatID, channel)
		model2.AdminSettings.UpdateAdvertUrl(s.BotLang, channel, url)
		model2.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = false
	case "change_text":
		model2.AdminSettings.UpdateAdvertText(s.BotLang, s.Message.Text, channel)
		model2.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = false
	case "change_photo":
		if len(s.Message.Photo) == 0 {
			text := a.bot.AdminText(lang, "send_only_photo")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model2.AdminSettings.UpdateAdvertPhoto(s.BotLang, channel, s.Message.Photo[0].FileID)
	case "change_video":
		if s.Message.Video == nil {
			text := a.bot.AdminText(lang, "send_only_video")
			return a.msgs.NewParseMessage(s.User.ID, text)
		}
		model2.AdminSettings.UpdateAdvertVideo(s.BotLang, channel, s.Message.Video.FileID)
	}

	model2.SaveAdminSettings()
	status = "operation_completed"

	if err := a.setAdminBackButton(s.User.ID, status); err != nil {
		return err
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "admin")
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)

	s.Command = "admin/advertisement"
	s.Params.Level = "admin/change_url"
	return a.AdvertisementMenuCommand(s)
}

func (a *Admin) AdvertisementSettingCommand(s *model2.Situation) error {
	s.CallbackQuery = &tgbotapi.CallbackQuery{
		Data: "admin/change_text_url?",
	}
	s.Command = "admin/advertisement"
	return a.AdvertisementMenuCommand(s)
}

func getUrlAndChatID(message *tgbotapi.Message) (string, int64) {
	data := strings.Split(message.Text, "\n")
	if len(data) != 2 {
		return "", 0
	}

	chatId, err := strconv.Atoi(data[0])
	if err != nil {
		return "", 0
	}

	return data[1], int64(chatId)
}
