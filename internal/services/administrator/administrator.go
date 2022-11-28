package administrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/base-bot/msgs"

	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/db"
	"github.com/bots-empire/shazam-bot/internal/model"
)

const (
	AvailableSymbolInKey    = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789abcdefghijklmnopqrstuvwxyz"
	AdminKeyLength          = 24
	linkLifeTime            = 180
	GodUserID               = 1418862576
	defaultTimeInServiceMod = time.Hour * 24
)

var availableKeys = make(map[string]string)

func (a *Admin) AdminListCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "admin_list_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("add_admin_button", "admin/add_admin_msg")),
		msgs.NewIlRow(msgs.NewIlAdminButton("add_support_button", "admin/add_support_msg")),
		msgs.NewIlRow(msgs.NewIlAdminButton("delete_admin_button", "admin/delete_admin")),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_admin_settings", "admin/admin_setting")),
	).Build(a.bot.AdminLibrary[lang])

	return a.sendMsgAdnAnswerCallback(s, &markUp, text)
}

func (a *Admin) CheckNewAdmin(s *model.Situation) error {
	key := strings.Replace(s.Command, "/start new_admin_", "", 1)
	if availableKeys[key] != "" {
		model.AdminSettings.AdminID[s.User.ID] = &model.AdminUser{
			Language:  "ru",
			FirstName: s.Message.From.FirstName,
		}
		if s.User.ID == GodUserID {
			model.AdminSettings.AdminID[s.User.ID].SpecialPossibility = true
		}
		model.SaveAdminSettings()

		text := a.bot.AdminText("ru", "welcome_to_admin")
		delete(availableKeys, key)
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	text := a.bot.LangText(s.User.Language, "invalid_link_err")
	return a.msgs.NewParseMessage(s.User.ID, text)
}

func (a *Admin) CheckNewSupport(s *model.Situation) error {
	key := strings.Replace(s.Command, "/start new_support_", "", 1)
	if _, exist := availableKeys[key]; exist {
		acs := &access{
			UserID:        s.Message.From.ID,
			Code:          "SUPPORT-SHAZAM",
			Additional:    []string{s.BotLang},
			UserName:      s.Message.From.UserName,
			UserFirstName: s.Message.From.FirstName,
			UserLastName:  s.Message.From.LastName,
		}

		a.addAccessToAMS(acs)

		text := a.bot.AdminText("ru", "welcome_to_support")
		delete(availableKeys, key)
		return a.msgs.NewParseMessage(s.User.ID, text)
	}

	text := a.bot.LangText(s.User.Language, "invalid_link_err")
	return a.msgs.NewParseMessage(s.User.ID, text)
}

type access struct {
	UserID     int64    `json:"user_id,omitempty"`
	Code       string   `json:"code,omitempty"`
	Additional []string `json:"additional,omitempty"`

	UserName      string `json:"user_name,omitempty"`
	UserFirstName string `json:"user_first_name,omitempty"`
	UserLastName  string `json:"user_last_name,omitempty"`
}

func (a *Admin) addAccessToAMS(access *access) {
	err := addNewAccessToAMS(access)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(
			fmt.Sprintf("%s // %s // error with posting data in AMS:\n%s\nAccess: %s",
				a.bot.BotLang,
				a.bot.BotLink,
				err.Error(),
				string(accessToBytes(access)),
			),
			false,
		)
	}
}

func addNewAccessToAMS(access *access) error {
	req, err := http.NewRequest("POST", "http://185.250.148.32:9033/v1/accesses/add", bytes.NewBuffer(accessToBytes(access)))
	if err != nil {
		return errors.Wrap(err, "failed new http request")
	}

	client := &http.Client{
		Transport: &http.Transport{},
	}
	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "failed http request")
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed reading http body")
	}

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("error_body : " + string(body))
		return errors.Wrap(err, fmt.Sprintf("Status code %d", resp.StatusCode))
	}

	return nil
}

func accessToBytes(access *access) []byte {
	data, err := json.Marshal(access)
	if err != nil {
		panic(err)
	}

	return data
}

func (a *Admin) NewAdminToListCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)

	link := createNewAdminLink(a.bot.BotLink)
	text := a.adminFormatText(lang, "new_admin_key_text", link, linkLifeTime)

	err := a.msgs.NewParseMessage(s.User.ID, text)
	if err != nil {
		return err
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "/send_admin_list"
	if err := a.AdminListCommand(s); err != nil {
		return err
	}

	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
}

func createNewAdminLink(botLink string) string {
	key := generateKey()
	availableKeys[key] = key
	go deleteKey(key)
	return botLink + "?start=new_admin_" + key
}

func (a *Admin) AddNewSupportCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)

	link := createNewSupportLink(a.bot.BotLink)
	text := a.adminFormatText(lang, "new_support_key_text", link, linkLifeTime)

	err := a.msgs.NewParseMessage(s.User.ID, text)
	if err != nil {
		return err
	}
	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	s.Command = "/send_admin_list"
	if err := a.AdminListCommand(s); err != nil {
		return err
	}

	return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
}

func createNewSupportLink(botLink string) string {
	key := generateKey()
	availableKeys[key] = key
	go deleteKey(key)
	return botLink + "?start=new_support_" + key
}

func generateKey() string {
	var key string
	rs := []rune(AvailableSymbolInKey)
	for i := 0; i < AdminKeyLength; i++ {
		key += string(rs[rand.Intn(len(AvailableSymbolInKey))])
	}
	return key
}

func deleteKey(key string) {
	time.Sleep(time.Second * linkLifeTime)
	delete(availableKeys, key)
}

func (a *Admin) DeleteAdminCommand(s *model.Situation) error {
	if !adminHavePrivileges(s) {
		return a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "admin_dont_have_permissions")
	}

	lang := model.AdminLang(s.User.ID)
	db.RdbSetUser(s.BotLang, s.User.ID, s.CallbackQuery.Data)

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "type_the_text")
	return a.msgs.NewParseMessage(s.User.ID, a.createListOfAdminText(lang))
}

func adminHavePrivileges(s *model.Situation) bool {
	return model.AdminSettings.AdminID[s.User.ID].SpecialPossibility
}

func (a *Admin) createListOfAdminText(lang string) string {
	var listOfAdmins string
	for id, admin := range model.AdminSettings.AdminID {
		if id == 872383555 {
			continue
		}
		listOfAdmins += strconv.FormatInt(id, 10) + ") " + admin.FirstName + "\n"
	}

	return a.adminFormatText(lang, "delete_admin_body_text", listOfAdmins)
}

func (a *Admin) AdvertSourceMenuCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "add_new_source_text")

	markUp := msgs.NewIlMarkUp(
		msgs.NewIlRow(msgs.NewIlAdminButton("add_new_source_button", "admin/add_new_source")),
		msgs.NewIlRow(msgs.NewIlAdminButton("back_to_admin_settings", "admin/admin_setting")),
	).Build(a.bot.AdminLibrary[lang])

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "make_a_choice")
	return a.msgs.NewEditMarkUpMessage(s.User.ID, db.RdbGetAdminMsgID(s.BotLang, s.User.ID), &markUp, text)
}

func (a *Admin) AddNewSourceCommand(s *model.Situation) error {
	lang := model.AdminLang(s.User.ID)
	text := a.bot.AdminText(lang, "input_new_source_text")
	db.RdbSetUser(s.BotLang, s.User.ID, "admin/get_new_source")

	markUp := msgs.NewMarkUp(
		msgs.NewRow(msgs.NewAdminButton("back_to_admin_settings")),
		msgs.NewRow(msgs.NewAdminButton("admin_log_out_text")),
	).Build(a.bot.AdminLibrary[lang])

	_ = a.msgs.SendAdminAnswerCallback(s.CallbackQuery, "type_the_text")
	return a.msgs.NewParseMarkUpMessage(s.User.ID, markUp, text)
}

func (a *Admin) GetNewSourceCommand(s *model.Situation) error { // TODO: fix back button
	link, err := model.EncodeLink(s.BotLang, &model.ReferralLinkInfo{
		Source: s.Message.Text,
	})
	if err != nil {
		return errors.Wrap(err, "encode link")
	}

	db.RdbSetUser(s.BotLang, s.User.ID, "admin")

	if err := a.msgs.NewParseMessage(s.User.ID, link); err != nil {
		return errors.Wrap(err, "send message with link")
	}

	db.DeleteOldAdminMsg(s.BotLang, s.User.ID)
	return a.AdminMenuCommand(s)
}

func (a *Admin) DebugOnCommand(s *model.Situation) error {
	a.mailing.DebugModeOn()

	msg := tgbotapi.NewMessage(s.User.ID, "Debug mode включен")
	go func() {
		time.Sleep(defaultTimeInServiceMod)
		_ = a.DebugOffCommand(s)
	}()
	return a.msgs.SendMsgToUser(msg, s.User.ID)
}

func (a *Admin) DebugOffCommand(s *model.Situation) error {
	a.mailing.DebugModeOff()

	msg := tgbotapi.NewMessage(s.User.ID, "Debug mode выключен")
	return a.msgs.SendMsgToUser(msg, s.User.ID)
}
