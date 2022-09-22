package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/roylee0704/gron"
	"github.com/roylee0704/gron/xtime"

	"github.com/bots-empire/base-bot/msgs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/db"
	"github.com/bots-empire/shazam-bot/internal/log"
	model2 "github.com/bots-empire/shazam-bot/internal/model"
	"github.com/bots-empire/shazam-bot/internal/services/administrator"
	"github.com/bots-empire/shazam-bot/internal/utils"
)

const (
	updateCounterHeader = "Today Update's counter: %d"
	updatePrintHeader   = "update number: %d    // voice-bot-update:  %s %s"
	godUserID           = 1418862576

	defaultTimeInServiceMod = time.Hour * 24
)

type MessagesHandlers struct {
	Handlers map[string]model2.Handler
}

func (h *MessagesHandlers) GetHandler(command string) model2.Handler {
	return h.Handlers[command]
}

func (h *MessagesHandlers) Init(userSrv *Users, adminSrv *administrator.Admin) {
	//Start command
	h.OnCommand("/select_language", userSrv.SelectLangCommand)
	h.OnCommand("/start", userSrv.StartCommand)
	h.OnCommand("/admin", adminSrv.AdminLoginCommand)

	//Main command
	h.OnCommand("/main_profile", userSrv.SendProfileCommand)
	h.OnCommand("/main_money_for_a_friend", userSrv.MoneyForAFriendCommand)
	h.OnCommand("/main_more_money", userSrv.MoreMoneyCommand)
	h.OnCommand("/main_make_money", userSrv.MakeMoneyCommand)
	h.OnCommand("/new_make_money", userSrv.MakeMoneyMsgCommand)
	h.OnCommand("/main_statistic", userSrv.MakeStatisticCommand)

	//Spend money command
	h.OnCommand("/main_withdrawal_of_money", userSrv.SpendMoneyWithdrawalCommand)
	h.OnCommand("/paypal_method", userSrv.PaypalReqCommand)
	h.OnCommand("/credit_card_method", userSrv.CreditCardReqCommand)
	h.OnCommand("/withdrawal_method", userSrv.WithdrawalMethodCommand)
	h.OnCommand("/withdrawal_req_amount", userSrv.ReqWithdrawalAmountCommand)
	h.OnCommand("/withdrawal_exit", userSrv.WithdrawalAmountCommand)
	h.OnCommand("/main_top_players", userSrv.TopListPlayerCommand)

	//Log out command
	h.OnCommand("/admin_log_out", userSrv.AdminLogOutCommand)

	//Tech command
	h.OnCommand("/mmon", userSrv.MaintenanceModeOnCommand)
	h.OnCommand("/mmoff", userSrv.MaintenanceModeOffCommand)
}

func (h *MessagesHandlers) OnCommand(command string, handler model2.Handler) {
	h.Handlers[command] = handler
}

func (u *Users) ActionsWithUpdates(logger log.Logger, sortCentre *utils.Spreader, cron *gron.Cron) {
	//start top handler
	cron.AddFunc(gron.Every(1*xtime.Day).At("15:05"), u.TopListPlayers)

	for update := range u.bot.Chanel {
		localUpdate := update

		go u.checkUpdate(&localUpdate, logger, sortCentre)
	}
}

func (u *Users) checkUpdate(update *tgbotapi.Update, logger log.Logger, sortCentre *utils.Spreader) {
	defer u.panicCather(update)

	if update.Message == nil && update.CallbackQuery == nil {
		return
	}

	if update.Message != nil && update.Message.PinnedMessage != nil {
		return
	}

	u.printNewUpdate(update, logger)
	if update.Message != nil {
		var command string
		user, err := u.auth.CheckingTheUser(update.Message)
		if err == model2.ErrNotSelectedLanguage {
			command = "/select_language"
		} else if err != nil {
			u.smthWentWrong(update.Message.Chat.ID, u.bot.BotLang)
			logger.Warn("err with check user: %s", err.Error())
			return
		}

		situation := createSituationFromMsg(u.bot.BotLang, update.Message, user)
		situation.Command = command

		u.checkMessage(&situation, logger, sortCentre)
		return
	}

	if update.CallbackQuery != nil {
		if strings.Contains(update.CallbackQuery.Data, "/language") {
			err := u.auth.SetStartLanguage(update.CallbackQuery)
			if err != nil {
				u.smthWentWrong(update.CallbackQuery.Message.Chat.ID, u.bot.BotLang)
				logger.Warn("err with set start language: %s", err.Error())
			}
		}
		situation, err := u.createSituationFromCallback(u.bot.BotLang, update.CallbackQuery)
		if err != nil {
			u.smthWentWrong(update.CallbackQuery.Message.Chat.ID, u.bot.BotLang)
			logger.Warn("err with create situation from callback: %s", err.Error())
			return
		}

		u.checkCallbackQuery(situation, logger, sortCentre)
		return
	}
}

func (u *Users) printNewUpdate(update *tgbotapi.Update, logger log.Logger) {
	model2.UpdateStatistic.Mu.Lock()
	defer model2.UpdateStatistic.Mu.Unlock()

	if (time.Now().Unix())/86400 > int64(model2.UpdateStatistic.Day) {
		u.sendTodayUpdateMsg()
	}

	model2.UpdateStatistic.Counter++
	model2.SaveUpdateStatistic()

	model2.HandleUpdates.WithLabelValues(
		u.bot.BotLink,
		u.bot.BotLang,
	).Inc()

	if update.Message != nil {
		if update.Message.Text != "" {
			logger.Info(updatePrintHeader, model2.UpdateStatistic.Counter, u.bot.BotLang, update.Message.Text)
			return
		}
	}

	if update.CallbackQuery != nil {
		logger.Info(updatePrintHeader, model2.UpdateStatistic.Counter, u.bot.BotLang, update.CallbackQuery.Data)
		return
	}

	//logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, extraneousUpdate)
}

func (u *Users) sendTodayUpdateMsg() {
	text := fmt.Sprintf(updateCounterHeader, model2.UpdateStatistic.Counter)
	u.Msgs.SendNotificationToDeveloper(text, true)

	model2.UpdateStatistic.Counter = 0
	model2.UpdateStatistic.Day = int(time.Now().Unix()) / 86400
}

func createSituationFromMsg(botLang string, message *tgbotapi.Message, user *model2.User) model2.Situation {
	return model2.Situation{
		Message: message,
		BotLang: botLang,
		User:    user,
		Params: &model2.Parameters{
			Level: db.GetLevel(botLang, message.From.ID),
		},
	}
}

func (u *Users) createSituationFromCallback(botLang string, callbackQuery *tgbotapi.CallbackQuery) (*model2.Situation, error) {
	user, err := u.auth.GetUser(callbackQuery.From.ID)
	if err != nil {
		return &model2.Situation{}, err
	}

	return &model2.Situation{
		CallbackQuery: callbackQuery,
		BotLang:       botLang,
		User:          user,
		Command:       strings.Split(callbackQuery.Data, "?")[0],
		Params: &model2.Parameters{
			Level: db.GetLevel(botLang, callbackQuery.From.ID),
		},
	}, nil
}

func (u *Users) checkMessage(situation *model2.Situation, logger log.Logger, sortCentre *utils.Spreader) {
	maintenanceMode := model2.AdminSettings.UnderMaintenance(situation.BotLang)

	if situation.Command == "" {
		situation.Command, situation.Err = u.bot.GetCommandFromText(
			situation.Message, situation.User.Language, situation.User.ID)
	}

	if situation.Err == nil && (!maintenanceMode || isTechCommand(situation.Command)) {
		handler := model2.Bots[situation.BotLang].MessageHandler.
			GetHandler(situation.Command)

		if handler != nil {
			sortCentre.ServeHandler(handler, situation, func(err error) {
				text := fmt.Sprintf("%s // %s // error with serve user msg command: %s\ncommand = '%s'",
					u.bot.BotLang,
					u.bot.BotLink,
					err.Error(),
					situation.Command,
				)
				u.Msgs.SendNotificationToDeveloper(text, false)

				logger.Warn(text)
				u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
			})
			return
		}
	}

	situation.Command = strings.Split(situation.Params.Level, "?")[0]

	handler := model2.Bots[situation.BotLang].MessageHandler.
		GetHandler(situation.Command)

	if handler != nil {
		sortCentre.ServeHandler(handler, situation, func(err error) {
			text := fmt.Sprintf("%s // %s // error with serve user level command: %s\ncommand = '%s'",
				u.bot.BotLang,
				u.bot.BotLink,
				err.Error(),
				situation.Command,
			)
			u.Msgs.SendNotificationToDeveloper(text, false)

			logger.Warn(text)
			u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
		})
		return
	}

	if err := u.admin.CheckAdminMessage(situation); err != nil {
		if err != model2.ErrCommandNotConverted {
			text := fmt.Sprintf(
				"%s // %s // error with serve admin level command: %s\ncommand = '%s'",
				u.bot.BotLang,
				u.bot.BotLink,
				err,
				situation.Command,
			)
			u.Msgs.SendNotificationToDeveloper(text, false)

			return
		}
	}

	if maintenanceMode {
		model2.LossUserMessages.WithLabelValues(situation.BotLang).Inc()
		return
	}

	u.smthWentWrong(situation.Message.Chat.ID, situation.User.Language)
	if situation.Err != nil {
		logger.Info(situation.Err.Error())
	}
}

var (
	techCommands = []string{"/mmoff", "/mmon", "/admin", "/admin_log_out"}
)

func isTechCommand(command string) bool {
	for _, techCommand := range techCommands {
		if command == techCommand {
			return true
		}
	}

	return false
}

func (u *Users) SendTodayUpdateMsg() {
	model2.UpdateStatistic.Mu.Lock()
	defer model2.UpdateStatistic.Mu.Unlock()

	text := fmt.Sprintf(updateCounterHeader, model2.UpdateStatistic.Counter)
	u.Msgs.SendNotificationToDeveloper(text, true)

	model2.UpdateStatistic.Counter = 0
}

func (u *Users) smthWentWrong(chatID int64, lang string) {
	msg := tgbotapi.NewMessage(chatID, u.bot.LangText(lang, "user_level_not_defined"))
	_ = u.Msgs.SendMsgToUser(msg, chatID)
}

func (u *Users) emptyLevel(message *tgbotapi.Message, lang string) {
	msg := tgbotapi.NewMessage(message.Chat.ID, u.bot.LangText(lang, "user_level_not_defined"))
	_ = u.Msgs.SendMsgToUser(msg, message.Chat.ID)
}

func createMainMenu() msgs.MarkUp {
	return msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("main_make_money")),
		msgs.NewRow(msgs.NewDataButton("main_profile"),
			msgs.NewDataButton("main_statistic")),
		msgs.NewRow(msgs.NewDataButton("main_withdrawal_of_money"),
			msgs.NewDataButton("main_money_for_a_friend")),
		msgs.NewRow(msgs.NewDataButton("main_more_money"),
			msgs.NewDataButton("main_top_players")),
	)
}

func (u *Users) SendProfileCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	text := u.bot.LangText(s.User.Language, "profile_text",
		s.Message.From.FirstName, s.Message.From.UserName, s.User.Balance, s.User.Completed, s.User.ReferralCount)

	if len(model2.GetGlobalBot(s.BotLang).LanguageInBot) > 1 {
		ReplyMarkup := u.createLangMenu(model2.GetGlobalBot(s.BotLang).LanguageInBot)
		return u.Msgs.NewParseMarkUpMessage(s.User.ID, &ReplyMarkup, text)
	}

	return u.Msgs.NewParseMessage(s.User.ID, text)
}

func (u *Users) MoneyForAFriendCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	link, err := model2.EncodeLink(s.BotLang, &model2.ReferralLinkInfo{
		ReferralID: s.User.ID,
		Source:     "bot",
	})
	if err != nil {
		return err
	}

	text := u.bot.LangText(s.User.Language, "referral_text",
		link,
		model2.AdminSettings.GetParams(s.BotLang).ReferralAmount,
		s.User.ReferralCount)

	return u.Msgs.NewParseMessage(s.User.ID, text)
}

func (u *Users) SelectLangCommand(s *model2.Situation) error {
	var text string
	for _, lang := range model2.GetGlobalBot(s.BotLang).LanguageInBot {
		text += u.bot.LangText(lang, "select_lang_menu") + "\n"
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = u.createLangMenu(model2.GetGlobalBot(s.BotLang).LanguageInBot)

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) createLangMenu(languages []string) tgbotapi.InlineKeyboardMarkup {
	var markup tgbotapi.InlineKeyboardMarkup

	for _, lang := range languages {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(u.bot.LangText(lang, "lang_button"), "/language?"+lang),
		})
	}

	return markup
}

func (u *Users) StartCommand(s *model2.Situation) error {
	if s.Message != nil {
		if strings.Contains(s.Message.Text, "new_admin") {
			s.Command = s.Message.Text
			return u.admin.CheckNewAdmin(s)
		}
	}

	text := u.bot.LangText(s.User.Language, "main_select_menu")
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = createMainMenu().Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) SpendMoneyWithdrawalCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "withdrawal")

	text := u.bot.LangText(s.User.Language, "select_payment")
	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_1"),
			msgs.NewDataButton("withdrawal_method_2")),
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_3"),
			msgs.NewDataButton("withdrawal_method_4")),
		msgs.NewRow(msgs.NewDataButton("withdrawal_method_5")),
		msgs.NewRow(msgs.NewDataButton("main_back")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

func (u *Users) PaypalReqCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "paypal_method"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) CreditCardReqCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "credit_card_number"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) WithdrawalMethodCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) ReqWithdrawalAmountCommand(s *model2.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_exit")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) WithdrawalAmountCommand(s *model2.Situation) error {
	return u.auth.WithdrawMoneyFromBalance(s, s.Message.Text)
}

func (u *Users) AdminLogOutCommand(s *model2.Situation) error {
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	if err := u.simpleAdminMsg(s, "admin_log_out"); err != nil {
		return err
	}

	return u.StartCommand(s)
}

func (u *Users) MakeStatisticCommand(s *model2.Situation) error {
	currentTime := time.Now()

	users := currentTime.Unix() % 100000000 / 6000
	totalEarned := currentTime.Unix() % 100000000 / 500 * 5
	totalVoice := totalEarned / 7

	text := u.bot.LangText(s.User.Language, "statistic_to_user", users, totalEarned, totalVoice)

	return u.Msgs.NewParseMessage(s.Message.Chat.ID, text)
}

func (u *Users) MakeMoneyCommand(s *model2.Situation) error {
	if !u.auth.MakeMoney(s) {
		text := u.bot.LangText(s.User.Language, "main_select_menu")
		msg := tgbotapi.NewMessage(s.User.ID, text)
		msg.ReplyMarkup = createMainMenu().Build(u.bot.Language[s.User.Language])

		return u.Msgs.SendMsgToUser(msg, s.User.ID)
	}

	return nil
}

func (u *Users) MakeMoneyMsgCommand(s *model2.Situation) error {
	if s.Message.Voice == nil {
		msg := tgbotapi.NewMessage(s.Message.Chat.ID, u.bot.LangText(s.User.Language, "voice_not_recognized"))
		_ = u.Msgs.SendMsgToUser(msg, s.User.ID)
		return nil
	}

	if !u.auth.AcceptVoiceMessage(s) {
		return nil
	}
	return nil
}

func (u *Users) MoreMoneyCommand(s *model2.Situation) error {
	model2.MoreMoneyButtonClick.WithLabelValues(
		model2.GetGlobalBot(s.BotLang).BotLink,
		s.BotLang,
	).Inc()

	db.RdbSetUser(s.BotLang, s.User.ID, "main")
	text := u.bot.LangText(s.User.Language, "more_money_text",
		model2.AdminSettings.GetParams(s.BotLang).BonusAmount, model2.AdminSettings.GetParams(s.BotLang).BonusAmount)

	markup := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlURLButton("advertising_button", model2.AdminSettings.GlobalParameters[s.BotLang].AdvertisingChan.Url[model2.MainAdvert])),
		msgs.NewIlRow(msgs.NewIlDataButton("get_bonus_button", "/send_bonus_to_user")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markup, text)
}

func (u *Users) MaintenanceModeOnCommand(s *model2.Situation) error {
	if s.User.ID != godUserID {
		return model2.ErrNotAdminUser
	}

	model2.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = true

	msg := tgbotapi.NewMessage(s.User.ID, "Режим технического обслуживания включен")
	go func() {
		time.Sleep(defaultTimeInServiceMod)
		_ = u.MaintenanceModeOffCommand(s)
	}()
	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) MaintenanceModeOffCommand(s *model2.Situation) error {
	if s.User.ID != godUserID {
		return model2.ErrNotAdminUser
	}

	model2.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = false

	msg := tgbotapi.NewMessage(s.User.ID, "Режим технического обслуживания отключен")
	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) simpleAdminMsg(s *model2.Situation, key string) error {
	lang := model2.AdminLang(s.User.ID)
	text := u.bot.AdminText(lang, key)
	msg := tgbotapi.NewMessage(s.User.ID, text)

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}
