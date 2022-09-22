package services

import (
	"database/sql"

	model2 "github.com/bots-empire/shazam-bot/internal/model"
)

func (u *Users) CreateNilTop(number int) error {
	dataBase := u.bot.GetDataBase()
	_, err := dataBase.Exec(`INSERT INTO shazam.top VALUES ($1,$2,$3,$4)`, number, 0, 0, 0)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) GetUserBalanceFromID(id int64) (int, error) {
	var balance int
	dataBase := u.bot.GetDataBase()
	err := dataBase.QueryRow(`
SELECT balance FROM shazam.users WHERE id = ?`, id).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (u *Users) GetUsers(limit int) ([]*model2.User, error) {
	dataBase := u.bot.GetDataBase()
	rows, err := dataBase.Query(`
SELECT id, balance FROM shazam.users ORDER BY balance DESC LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}

	user, err := readUserBalance(rows)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func readUserBalance(rows *sql.Rows) ([]*model2.User, error) {
	defer rows.Close()

	var users []*model2.User

	for rows.Next() {
		var id int64
		var balance int

		if err := rows.Scan(&id, &balance); err != nil {
			return nil, model2.ErrScanSqlRow
		}

		users = append(users, &model2.User{
			ID:      id,
			Balance: balance,
		})
	}
	if len(users) == 0 {
		users = append(users, &model2.User{
			ID:      0,
			Balance: 0,
		})
	}
	return users, nil
}

func (u *Users) GetFromTop(topNumber int) (*model2.Top, error) {
	dataBase := u.bot.GetDataBase()

	top := &model2.Top{
		Top: topNumber,
	}

	_ = dataBase.QueryRow(`SELECT user_id, time_on_top, balance FROM shazam.top WHERE top = ?;`,
		topNumber).Scan(&top.UserID, &top.TimeOnTop, &top.Balance)

	return top, nil
}

func (u *Users) GetTop() ([]*model2.Top, error) {
	dataBase := u.bot.GetDataBase()

	rows, err := dataBase.Query(`SELECT * FROM shazam.top;`)
	if err != nil {
		return nil, err
	}

	top, err := u.ReadRows(rows)
	if err != nil {
		return nil, err
	}

	return top, nil
}

func (u *Users) UpdateTop3Players(id int64, timeOnTop, topNumber, balance int) error {
	dataBase := u.bot.GetDataBase()

	_, err := dataBase.Exec(`UPDATE shazam.top SET user_id = $1, time_on_top = $2, balance = $3 WHERE top = $4;`, id, timeOnTop, balance, topNumber)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) UpdateTop3Balance(id int64, balance int) error {
	dataBase := u.bot.GetDataBase()

	_, err := dataBase.Exec(`
UPDATE shazam.users SET balance = $1
	WHERE id = $2;`, balance, id)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) ReadRows(rows *sql.Rows) ([]*model2.Top, error) {
	defer rows.Close()
	var topArr []*model2.Top

	for rows.Next() {
		top := &model2.Top{}
		err := rows.Scan(&top.Top, &top.UserID, &top.TimeOnTop, &top.Balance)
		if err != nil {
			return nil, err
		}

		topArr = append(topArr, top)
	}

	return topArr, nil
}
