package administrator

import (
	"database/sql"
	"math/rand"

	"github.com/pkg/errors"

	"github.com/bots-empire/shazam-bot/internal/model"
)

func (a *Admin) GetTask() (*model.ShazamTask, error) {
	rows, err := a.bot.GetDataBase().Query(`
SELECT * FROm shazam.tasks;`)
	if err != nil {
		return nil, errors.Wrap(err, "failed get tasks")
	}
	defer rows.Close()

	tasks, err := a.readTasks(rows)
	if err != nil {
		return nil, errors.Wrap(err, "failed read tasks from rows")
	}

	if len(tasks) == 0 {
		return nil, model.ErrTaskNotFound
	}

	return tasks[rand.Intn(len(tasks))], nil
}

func (a *Admin) readTasks(rows *sql.Rows) ([]*model.ShazamTask, error) {
	var tasks []*model.ShazamTask

	for rows.Next() {
		task := &model.ShazamTask{}

		if err := rows.Scan(
			&task.FileID,
			&task.VoiceLength); err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}
	return tasks, nil
}
