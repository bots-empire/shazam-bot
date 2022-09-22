package administrator

import "github.com/pkg/errors"

func (a *Admin) AddTaskToDB(fileID string, voiceLength int) error {
	_, err := a.bot.GetDataBase().Exec("INSERT INTO shazam.tasks (file_id,voice_length) VALUES ($1,$2)", fileID, voiceLength)
	if err != nil {
		return errors.Wrap(err, "failed to insert file_id and voice_length")
	}

	return nil
}

func (a *Admin) DeleteTaskFromDB(taskID string) error {
	_, err := a.bot.GetDataBase().Exec("DELETE FROM shazam.tasks WHERE id = $1", taskID)
	if err != nil {
		return errors.Wrap(err, "failed to delete task from db")
	}

	return nil
}
