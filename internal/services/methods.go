package services

import (
	"database/sql"

	model2 "github.com/bots-empire/shazam-bot/internal/model"
)

func (u *Users) CreateNilTop(number int) error {
	dataBase := u.bot.GetDataBase()
	_, err := dataBase.Exec(`INSERT INTO top VALUES (?,?,?,?)`, number, 0, 0, 0)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) GetUserBalanceFromID(id int64) (int, error) {
	var balance int
	dataBase := u.bot.GetDataBase()
	err := dataBase.QueryRow(`
SELECT balance FROM users WHERE id = ?`, id).Scan(&balance)
	if err != nil {
		return 0, err
	}

	return balance, nil
}

func (u *Users) GetUsers(limit int) ([]*model2.User, error) {
	dataBase := u.bot.GetDataBase()
	rows, err := dataBase.Query(`
SELECT id, balance FROM users ORDER BY balance DESC LIMIT ?`, limit)
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

	_ = dataBase.QueryRow(`SELECT user_id, time_on_top, balance FROM top WHERE top = ?;`,
		topNumber).Scan(&top.UserID, &top.TimeOnTop, &top.Balance)

	return top, nil
}

func (u *Users) GetTop() ([]*model2.Top, error) {
	dataBase := u.bot.GetDataBase()

	rows, err := dataBase.Query(`SELECT * FROM top;`)
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

	_, err := dataBase.Exec(`UPDATE top SET user_id = ?, time_on_top = ?, balance = ? WHERE top = ?;`, id, timeOnTop, balance, topNumber)
	if err != nil {
		return err
	}

	return nil
}

func (u *Users) UpdateTop3Balance(id int64, balance int) error {
	dataBase := u.bot.GetDataBase()

	_, err := dataBase.Exec(`
UPDATE users SET balance = ?
	WHERE id = ?;`, balance, id)
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
