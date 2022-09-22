package administrator

import (
	"database/sql"
	"log"
	"strconv"

	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/model"
)

func (a *Admin) CountUsers() int {
	rows, err := a.bot.GetDataBase().Query(`
SELECT COUNT(*) FROM shazam.users;`)
	if err != nil {
		log.Println(err)
		return 0
	}
	count, err := readRows(rows)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(err.Error(), false)
	}

	return count
}

func readRows(rows *sql.Rows) (int, error) {
	defer rows.Close()

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return 0, errors.Wrap(err, "failed to scan row")
		}
	}

	return count, nil
}

func (a *Admin) countAllUsers() int {
	var sum int
	for _, handler := range model.Bots {
		rows, err := handler.DataBase.Query(`
SELECT COUNT(*) FROM shazam.users;`)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		count, err := readRows(rows)
		if err != nil {
			a.msgs.SendNotificationToDeveloper(err.Error(), false)
		}

		sum += count
	}
	return sum
}

func (a *Admin) countReferrals(botLang string, amountUsers int) string {
	var refText string
	rows, err := model.Bots[botLang].DataBase.Query("SELECT SUM(referral_count) FROM shazam.users;")
	if err != nil {
		log.Println(err.Error())
		return ""
	}

	count, err := readRows(rows)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(err.Error(), false)
	}

	refText = strconv.Itoa(count) + " (" + strconv.Itoa(int(float32(count)*100.0/float32(amountUsers))) + "%)"
	return refText
}

func (a *Admin) countBlockedUsers(botLang string) int {
	rows, err := model.Bots[botLang].DataBase.Query(`
SELECT COUNT(DISTINCT id) FROM shazam.users WHERE status = 'deleted';`)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(err.Error(), false)
		return 0
	}

	count, err := readRows(rows)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(err.Error(), false)
	}
	return count
}

func (a *Admin) countSubscribers(botLang string) int {
	rows, err := model.Bots[botLang].DataBase.Query(`
SELECT COUNT(DISTINCT id) FROM shazam.subs;`)
	if err != nil {
		log.Println(err)
		return 0
	}

	count, err := readRows(rows)
	if err != nil {
		a.msgs.SendNotificationToDeveloper(err.Error(), false)
	}

	return count
}
