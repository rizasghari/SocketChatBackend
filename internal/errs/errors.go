package errs

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidRequestBody = Error("invalid request body")
	ErrUserAlreadyExists  = Error("user already exists")
	ErrUserNotFound       = Error("user not found")
	ErrUserIsNil          = Error("user is nil")
	ErrWrongPassword      = Error("wrong password")
	ErrWrongEmail         = Error("wrong email")
	ErrWrongToken         = Error("wrong token")
	ErrInvalidToken       = Error("invalid token")
	ErrInvalidEmail       = Error("invalid email")
	ErrInvalidPassword    = Error("invalid password")
	ErrInvalidUser        = Error("invalid user")
	ErrInvalidRequest     = Error("invalid request")
	ErrInvalidParams      = Error("invalid params")
	ErrFirstName          = Error("first name is empty or too short")
	ErrLastName           = Error("last name is empty or too short")
)
