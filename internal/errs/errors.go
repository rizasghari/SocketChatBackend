package errs

type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrInvalidPassword            = Error("invalid password")
	ErrPasswordAtLeast8Characters = Error("password must be at least 8 characters long")
	ErrPasswordAtLeastOneDigit    = Error("password must contain at least one digit")
	ErrPasswordAtLeastOneSpecial  = Error("password must contain at least one special character")
	ErrPasswordAtLeastOneLower    = Error("password must contain at least one lowercase letter")
	ErrPasswordAtLeastOneUpper    = Error("password must contain at least one uppercase letter")

	ErrUnauthorized = Error("unauthorized")

	ErrConversationCreationFailed = Error("conversation creation failed")
	ErrInvalidConversationId      = Error("invalid conversation id")

	ErrNoFileUploaded           = Error("no file uploaded")
	ErrUnableToOpenUploadedFile = Error("unable to open uploaded file")
	ErrUnableToUploadFile       = Error("unable to upload file")
	ErrUnableToUpdateProfilePhoto = Error("unable to update profile photo")

	ErrMessageNotFound = Error("message not found")

	ErrConversationNotFound = Error("conversation not found")
	NoneOfMessagesSeen      = Error("none of messages seen")

	ErrInvalidRequestBody = Error("invalid request body")
	ErrUserAlreadyExists  = Error("user already exists")
	ErrUserNotFound       = Error("user not found")
	ErrThereIsNoUser      = Error("there is no user")
	ErrUserIsNil          = Error("user is nil")
	ErrWrongPassword      = Error("wrong password")
	ErrWrongEmail         = Error("wrong email")
	ErrWrongToken         = Error("wrong token")
	ErrInvalidToken       = Error("invalid token")
	ErrInvalidEmail       = Error("invalid email")
	ErrInvalidUser        = Error("invalid user")
	ErrInvalidRequest     = Error("invalid request")
	ErrInvalidParams      = Error("invalid params")
	ErrInvalidPageOrSize  = Error("invalid page or size")
	ErrFirstName          = Error("first name is empty or too short")
	ErrLastName           = Error("last name is empty or too short")
)
