package model

import (
	"database/sql"
	"math/rand"

	"github.com/pkg/errors"
)

type ShazamTask struct {
	FileID      string
	VoiceLength int
}

func GetAllTasks(db *sql.DB) ([]*ShazamTask, error) {
	rows, err := db.Query(`
SELECT * FROm shazam.tasks;`)
	if err != nil {
		return nil, errors.Wrap(err, "get tasks from query")
	}
	defer rows.Close()

	tasks, err := readTasks(rows)
	if err != nil {
		return nil, errors.Wrap(err, "read tasks from rows")
	}

	return tasks, nil
}

func GetTask(db *sql.DB) (*ShazamTask, error) {
	tasks, err := GetAllTasks(db)
	if err != nil {
		return nil, errors.Wrap(err, "get all tasks")
	}

	if len(tasks) == 0 {
		return nil, ErrTaskNotFound
	}

	return tasks[rand.Intn(len(tasks))], nil
}

func readTasks(rows *sql.Rows) ([]*ShazamTask, error) {
	var tasks []*ShazamTask

	for rows.Next() {
		task := &ShazamTask{}

		if err := rows.Scan(
			&task.FileID,
			&task.VoiceLength); err != nil {
			return nil, err
		}

		tasks = append(tasks, task)
	}
	return tasks, nil
}
