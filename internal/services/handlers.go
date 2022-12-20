package services

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/roylee0704/gron"
	"github.com/roylee0704/gron/xtime"

	"github.com/bots-empire/base-bot/msgs"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/db"
	"github.com/bots-empire/shazam-bot/internal/log"
	"github.com/bots-empire/shazam-bot/internal/model"
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
	Handlers map[string]model.Handler
}

func (h *MessagesHandlers) GetHandler(command string) model.Handler {
	return h.Handlers[command]
}

func (h *MessagesHandlers) Init(userSrv *Users, adminSrv *administrator.Admin) {
	//Start command
	h.OnCommand("/select_language", userSrv.SelectLangCommand)
	h.OnCommand("/start", userSrv.StartCommand)
	h.OnCommand("/admin", adminSrv.AdminLoginCommand)

	//Main command
	h.OnCommand("/new_make_money", userSrv.MakeMoneyMsgCommand)
	h.OnCommand("/withdrawal_req_amount", userSrv.ReqWithdrawalAmountCommand)
	h.OnCommand("/withdrawal_exit", userSrv.WithdrawalAmountCommand)

	//Log out command
	h.OnCommand("/admin_log_out", userSrv.AdminLogOutCommand)

	//Tech command
	h.OnCommand("/mmon", userSrv.MaintenanceModeOnCommand)
	h.OnCommand("/mmoff", userSrv.MaintenanceModeOffCommand)
	h.OnCommand("/debugon", userSrv.DebugOnCommand)
	h.OnCommand("/debugoff", userSrv.DebugOffCommand)
}

func (h *MessagesHandlers) OnCommand(command string, handler model.Handler) {
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

	st := time.Now()
	u.printNewUpdate(update, logger)
	if update.Message != nil {
		var command string
		user, err := u.auth.CheckingTheUser(update.Message)
		if err == model.ErrNotSelectedLanguage {
			command = "/select_language"
		} else if err != nil {
			u.smthWentWrong(update.Message.Chat.ID, u.bot.BotLang)
			logger.Warn("err with check user: %s", err.Error())
			return
		}

		situation := createSituationFromMsg(u.bot.BotLang, update.Message, user, st)
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
		situation, err := u.createSituationFromCallback(u.bot.BotLang, update.CallbackQuery, st)
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
	model.UpdateStatistic.Mu.Lock()
	defer model.UpdateStatistic.Mu.Unlock()

	model.UpdateStatistic.Counter++
	model.SaveUpdateStatistic()

	model.HandleUpdates.WithLabelValues(
		u.bot.BotLink,
		u.bot.BotLang,
	).Inc()

	if update.Message != nil {
		if update.Message.Text != "" {
			logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, update.Message.Text)
			return
		}
	}

	if update.CallbackQuery != nil {
		logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, update.CallbackQuery.Data)
		return
	}

	//logger.Info(updatePrintHeader, model.UpdateStatistic.Counter, u.bot.BotLang, extraneousUpdate)
}

func (u *Users) sendTodayUpdateMsg() {
	text := fmt.Sprintf(updateCounterHeader, model.UpdateStatistic.Counter)
	u.Msgs.SendNotificationToDeveloper(text, true)

	model.UpdateStatistic.Counter = 0
	model.UpdateStatistic.Day = int(time.Now().Unix()) / 86400
}

func createSituationFromMsg(botLang string, message *tgbotapi.Message, user *model.User, st time.Time) model.Situation {
	return model.Situation{
		Message: message,
		BotLang: botLang,
		User:    user,
		Params: &model.Parameters{
			Level: db.GetLevel(botLang, message.From.ID),
		},
		StartTime: st,
	}
}

func (u *Users) createSituationFromCallback(botLang string, callbackQuery *tgbotapi.CallbackQuery, st time.Time) (*model.Situation, error) {
	user, err := u.auth.GetUser(callbackQuery.From.ID)
	if err != nil {
		return &model.Situation{}, err
	}

	return &model.Situation{
		CallbackQuery: callbackQuery,
		BotLang:       botLang,
		User:          user,
		Command:       strings.Split(callbackQuery.Data, "?")[0],
		Params: &model.Parameters{
			Level: db.GetLevel(botLang, callbackQuery.From.ID),
		},
		StartTime: st,
	}, nil
}

func (u *Users) checkMessage(situation *model.Situation, logger log.Logger, sortCentre *utils.Spreader) {
	maintenanceMode := model.AdminSettings.UnderMaintenance(situation.BotLang)

	if situation.Command == "" {
		situation.Command, situation.Err = u.bot.GetCommandFromText(
			situation.Message, situation.User.Language, situation.User.ID)
	}

	if situation.Err == nil && (!maintenanceMode || isTechCommand(situation.Command)) {
		handler := model.Bots[situation.BotLang].MessageHandler.
			GetHandler(situation.Command)

		if handler != nil {
			sortCentre.ServeHandler(handler, situation, func(err error) {
				text := fmt.Sprintf("%s // error with serve user msg command: %s\ncommand = '%s'",
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

	handler := model.Bots[situation.BotLang].MessageHandler.
		GetHandler(situation.Command)

	if handler != nil {
		sortCentre.ServeHandler(handler, situation, func(err error) {
			text := fmt.Sprintf("%s // error with serve user level command: %s\ncommand = '%s'",
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

	err := u.admin.CheckAdminMessage(situation)
	if err == nil {
		return
	}
	text := fmt.Sprintf("%s // error with serve admin message command: %s\ncommand = '%s'",
		u.bot.BotLink,
		err,
		situation.Command,
	)
	u.Msgs.SendNotificationToDeveloper(text, false)

	if maintenanceMode {
		model.LossUserMessages.WithLabelValues(situation.BotLang).Inc()
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
	model.UpdateStatistic.Mu.Lock()
	defer model.UpdateStatistic.Mu.Unlock()

	text := fmt.Sprintf(updateCounterHeader, model.UpdateStatistic.Counter)
	u.Msgs.SendNotificationToDeveloper(text, true)

	model.UpdateStatistic.Counter = 0
}

func (u *Users) smthWentWrong(chatID int64, lang string) {
	msg := tgbotapi.NewMessage(chatID, u.bot.LangText(lang, "user_level_not_defined"))
	_ = u.Msgs.SendMsgToUser(msg, chatID)
}

func (u *Users) emptyLevel(message *tgbotapi.Message, lang string) {
	msg := tgbotapi.NewMessage(message.Chat.ID, u.bot.LangText(lang, "user_level_not_defined"))
	_ = u.Msgs.SendMsgToUser(msg, message.Chat.ID)
}

func createMainMenu() msgs.InlineMarkUp {
	return msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlDataButton("main_make_money", "/main_make_money")),
		msgs.NewIlRow(msgs.NewIlDataButton("main_money_for_a_friend", "/main_money_for_a_friend")),
		msgs.NewIlRow(msgs.NewIlDataButton("main_top_players", "/main_top_players")),
		msgs.NewIlRow(msgs.NewIlDataButton("main_withdrawal_of_money", "/main_withdrawal_of_money"),
			msgs.NewIlDataButton("main_profile", "/main_profile")),
		msgs.NewIlRow(msgs.NewIlDataButton("main_statistic", "/main_statistic"),
			msgs.NewIlDataButton("main_more_money", "/main_more_money")),
	)
}

func (u *Users) SendProfileCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	text := u.bot.LangText(s.User.Language, "profile_text",
		s.CallbackQuery.From.FirstName, s.CallbackQuery.From.UserName, s.User.Balance, s.User.Completed, s.User.ReferralCount)

	if len(model.GetGlobalBot(s.BotLang).LanguageInBot) > 1 {
		replyMarkup := u.createLangMenu(model.GetGlobalBot(s.BotLang).LanguageInBot, s.User.Language)
		return u.Msgs.NewParseMarkUpMessage(s.User.ID, &replyMarkup, text)
	}

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, u.mainMenuButton(s.User.Language), text)
}

func (u *Users) MoneyForAFriendCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	link, err := model.EncodeLink(s.BotLang, &model.ReferralLinkInfo{
		ReferralID: s.User.ID,
		Source:     "bot",
	})
	if err != nil {
		return err
	}

	countOfFirstLvl := getFirstLvlRef(s.User.AllReferrals)

	text := u.bot.LangText(s.User.Language, "referral_text",
		link,
		model.AdminSettings.GetParams(s.BotLang).ReferralReward.GetReward(1, countOfFirstLvl),
		s.User.ReferralCount,
	)

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, u.mainMenuButton(s.User.Language), text)
}

func getFirstLvlRef(rawLvls string) int {
	refByLvl := strings.Split(rawLvls, "/")
	if len(refByLvl) == 0 {
		return 0
	}

	count, _ := strconv.Atoi(refByLvl[0])
	return count
}

func (u *Users) SelectLangCommand(s *model.Situation) error {
	var text string
	for _, lang := range model.GetGlobalBot(s.BotLang).LanguageInBot {
		text += u.bot.LangText(lang, "select_lang_menu") + "\n"
	}
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	msg := tgbotapi.NewMessage(s.User.ID, text)
	msg.ReplyMarkup = u.createLangMenu(model.GetGlobalBot(s.BotLang).LanguageInBot, s.User.Language)

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) createLangMenu(languages []string, userLang string) tgbotapi.InlineKeyboardMarkup {
	var markup tgbotapi.InlineKeyboardMarkup

	for _, lang := range languages {
		markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(u.bot.LangText(lang, "lang_button"), "/language?"+lang),
		})
	}

	markup.InlineKeyboard = append(markup.InlineKeyboard, []tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardButtonData(u.bot.LangText(userLang, "back_to_main_menu_button"), "/main_menu"),
	})

	return markup
}

func (u *Users) StartCommand(s *model.Situation) error {
	if s.Message != nil {
		if strings.Contains(s.Message.Text, "new_admin") {
			s.Command = s.Message.Text
			return u.admin.CheckNewAdmin(s)
		}

		if strings.Contains(s.Message.Text, "new_support") {
			s.Command = s.Message.Text
			return u.admin.CheckNewSupport(s)
		}
	}

	err := u.setKeyboardMenuButton(s)
	if err != nil {
		return errors.Wrap(err, "failed set under keyboard menu button")
	}

	text := u.bot.LangText(s.User.Language, "main_select_menu")
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	markUp := createMainMenu().Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markUp, text)
}

func (u *Users) RestartCommand(s *model.Situation) error {
	text := u.bot.LangText(s.User.Language, "main_select_menu")
	db.RdbSetUser(s.BotLang, s.User.ID, "main")

	markup := createMainMenu().Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, &markup, text)
}

func (u *Users) SpendMoneyWithdrawalCommand(s *model.Situation) error {
	text := u.bot.LangText(s.User.Language, "select_payment")
	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlDataButton("withdrawal_method_1", "/credit_card_method"),
			msgs.NewIlDataButton("withdrawal_method_2", "/credit_card_method")),
		msgs.NewIlRow(msgs.NewIlDataButton("withdrawal_method_3", "/credit_card_method"),
			msgs.NewIlDataButton("withdrawal_method_4", "/credit_card_method")),
		msgs.NewIlRow(msgs.NewIlDataButton("withdrawal_method_5", "/credit_card_method")),
		msgs.NewIlRow(msgs.NewIlDataButton("back_to_main_menu_button", "/main_menu")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, &markUp, text)
}

func (u *Users) PaypalReqCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "paypal_method"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) CreditCardReqCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "credit_card_number"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) WithdrawalMethodCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_req_amount")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))
	msg.ReplyMarkup = msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("withdraw_cancel")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) ReqWithdrawalAmountCommand(s *model.Situation) error {
	db.RdbSetUser(s.BotLang, s.User.ID, "/withdrawal_exit")

	msg := tgbotapi.NewMessage(s.User.ID, u.bot.LangText(s.User.Language, "req_withdrawal_amount"))

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) WithdrawalAmountCommand(s *model.Situation) error {
	return u.auth.WithdrawMoneyFromBalance(s, s.Message.Text)
}

func (u *Users) AdminLogOutCommand(s *model.Situation) error {
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	if err := u.simpleAdminMsg(s, "admin_log_out"); err != nil {
		return err
	}

	return u.StartCommand(s)
}

func (u *Users) MakeStatisticCommand(s *model.Situation) error {
	currentTime := time.Now()

	users := currentTime.Unix() % 100000000 / 6000
	totalEarned := currentTime.Unix() % 100000000 / 500 * 5
	totalVoice := totalEarned / 7

	text := u.bot.LangText(s.User.Language, "statistic_to_user", users, totalEarned, totalVoice)

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, u.mainMenuButton(s.User.Language), text)
}

func (u *Users) MakeMoneyCommand(s *model.Situation) error {
	if !u.auth.MakeMoney(s) {
		text := u.bot.LangText(s.User.Language, "main_select_menu")
		msg := tgbotapi.NewMessage(s.User.ID, text)
		msg.ReplyMarkup = createMainMenu().Build(u.bot.Language[s.User.Language])

		return u.Msgs.SendMsgToUser(msg, s.User.ID)
	}

	return nil
}

func (u *Users) MakeMoneyMsgCommand(s *model.Situation) error {
	if s.Message.Voice == nil {
		msg := tgbotapi.NewMessage(s.Message.Chat.ID, u.bot.LangText(s.User.Language, "voice_not_recognized"))
		_ = u.Msgs.SendMsgToUser(msg, s.User.ID)
		return nil
	}

	length := db.RdbGetLengthOfTask(s.BotLang, s.User.ID) / 2
	if s.Message.Voice.Duration < length {
		msg := tgbotapi.NewMessage(s.Message.Chat.ID, u.bot.LangText(s.User.Language, "voice_length_too_small"))
		_ = u.Msgs.SendMsgToUser(msg, s.User.ID)
		return nil
	}

	if !u.auth.AcceptVoiceMessage(s) {
		return nil
	}
	return nil
}

func (u *Users) MoreMoneyCommand(s *model.Situation) error {
	model.MoreMoneyButtonClick.WithLabelValues(
		model.GetGlobalBot(s.BotLang).BotLink,
		s.BotLang,
	).Inc()

	db.RdbSetUser(s.BotLang, s.User.ID, "main")
	text := u.bot.LangText(s.User.Language, "more_money_text",
		model.AdminSettings.GetParams(s.BotLang).BonusAmount,
	)

	markup := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlURLButton("advertising_button", model.AdminSettings.GlobalParameters[s.BotLang].AdvertisingChan.Url[model.MainAdvert])),
		msgs.NewIlRow(msgs.NewIlDataButton("get_bonus_button", "/send_bonus_to_user")),
		msgs.NewIlRow(msgs.NewIlDataButton("back_to_main_menu_button", "/main_menu")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewEditMarkUpMessage(s.User.ID, s.CallbackQuery.Message.MessageID, &markup, text)
}

func (u *Users) MaintenanceModeOnCommand(s *model.Situation) error {
	if s.User.ID != godUserID {
		return model.ErrNotAdminUser
	}

	model.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = true

	msg := tgbotapi.NewMessage(s.User.ID, "Режим технического обслуживания включен")
	go func() {
		time.Sleep(defaultTimeInServiceMod)
		_ = u.MaintenanceModeOffCommand(s)
	}()
	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) MaintenanceModeOffCommand(s *model.Situation) error {
	if s.User.ID != godUserID {
		return model.ErrNotAdminUser
	}

	model.AdminSettings.GlobalParameters[s.BotLang].MaintenanceMode = false

	msg := tgbotapi.NewMessage(s.User.ID, "Режим технического обслуживания отключен")
	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) simpleAdminMsg(s *model.Situation, key string) error {
	lang := model.AdminLang(s.User.ID)
	text := u.bot.AdminText(lang, key)
	msg := tgbotapi.NewMessage(s.User.ID, text)

	return u.Msgs.SendMsgToUser(msg, s.User.ID)
}

func (u *Users) setKeyboardMenuButton(s *model.Situation) error {
	text := u.bot.LangText(s.User.Language, "menu_keyboard_button")

	markup := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewDataButton("menu_keyboard_button")),
	).Build(u.bot.Language[s.User.Language])

	return u.Msgs.NewParseMarkUpMessage(s.User.ID, &markup, text)
}

func (u *Users) mainMenuButton(lang string) *tgbotapi.InlineKeyboardMarkup {
	markup := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlDataButton("back_to_main_menu_button", "/main_menu")),
	).Build(u.bot.Language[lang])

	return &markup
}

func (u *Users) DebugOnCommand(s *model.Situation) error {
	return u.admin.DebugOnCommand(s)
}

func (u *Users) DebugOffCommand(s *model.Situation) error {
	return u.admin.DebugOffCommand(s)
}
