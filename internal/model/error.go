package model

const (
	// ErrCommandNotConverted error command not recognize.
	ErrCommandNotConverted = Error("command not converted")
	// ErrUserNotFound error user not found.
	ErrUserNotFound = Error("user not found")
	// ErrFoundTwoUsers error found two user account for one user.
	ErrFoundTwoUsers = Error("found two users")
	// ErrNotAdminUser error not admin user.
	ErrNotAdminUser = Error("not admin user")
	// ErrTaskNotFound error task not found.
	ErrTaskNotFound = Error("task not found")

	ErrNotSelectedLanguage = Error("not selected language")

	// ErrScanSqlRow error scan sql row.
	ErrScanSqlRow = Error("failed scan sql row")

	// ErrRedisNil error redis: nil.
	ErrRedisNil = Error("redis: nil")
)

type Error string

func (e Error) Error() string {
	return string(e)
}
