package db

import (
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"github.com/bots-empire/shazam-bot/internal/model"
)

const (
	emptyLevelName = "empty"
)

func RdbSetUser(botLang string, ID int64, level string) {
	userID := userIDToRdb(botLang, ID)
	_, err := model.Bots[botLang].Rdb.Set(userID, level, 0).Result()
	if err != nil {
		log.Println(err)
	}
}

func userIDToRdb(botLang string, userID int64) string {
	return botLang + ":user:" + strconv.FormatInt(userID, 10)
}

func GetLevel(botLang string, id int64) string {
	userID := userIDToRdb(botLang, id)
	have, err := model.Bots[botLang].Rdb.Exists(userID).Result()
	if err != nil {
		log.Println(err)
	}
	if have == 0 {
		return emptyLevelName
	}

	value, err := model.Bots[botLang].Rdb.Get(userID).Result()
	if err != nil {
		log.Println(err)
	}
	return value
}

func RdbSetAdminMsgID(botLang string, userID int64, msgID int) {
	adminMsgID := adminMsgIDToRdb(botLang, userID)
	_, err := model.Bots[botLang].Rdb.Set(adminMsgID, strconv.Itoa(msgID), 0).Result()
	if err != nil {
		log.Println(err)
	}
}

func adminMsgIDToRdb(botLang string, userID int64) string {
	return botLang + ":admin_msg_id:" + strconv.FormatInt(userID, 10)
}

func RdbGetAdminMsgID(botLang string, userID int64) int {
	adminMsgID := adminMsgIDToRdb(botLang, userID)
	result, err := model.Bots[botLang].Rdb.Get(adminMsgID).Result()
	if err != nil {
		log.Println(err)
	}
	msgID, _ := strconv.Atoi(result)
	return msgID
}

func DeleteOldAdminMsg(botLang string, userID int64) {
	adminMsgID := adminMsgIDToRdb(botLang, userID)
	result, err := model.Bots[botLang].Rdb.Get(adminMsgID).Result()
	if err != nil {
		log.Println(err)
	}

	if oldMsgID, _ := strconv.Atoi(result); oldMsgID != 0 {
		msg := tgbotapi.NewDeleteMessage(userID, oldMsgID)

		if _, err = model.Bots[botLang].Bot.Send(msg); err != nil {
			log.Println(err)
		}
		RdbSetAdminMsgID(botLang, userID, 0)
	}
}

func topLevelSettingToRdb(botLang string, userID int64) string {
	return botLang + ":top_level_setting:" + strconv.FormatInt(userID, 10)
}

func RdbSetTopLevelSetting(botLang string, userID int64, level int) {
	topLevel := topLevelSettingToRdb(botLang, userID)
	_, err := model.Bots[botLang].Rdb.Set(topLevel, strconv.Itoa(level), 0).Result()
	if err != nil {
		log.Println(err)
	}
}

func RdbGetTopLevelSetting(botLang string, userID int64) int {
	topLevel := topLevelSettingToRdb(botLang, userID)
	result, err := model.Bots[botLang].Rdb.Get(topLevel).Result()
	if err != nil {
		log.Println(err)
	}
	level, _ := strconv.Atoi(result)
	return level
}
