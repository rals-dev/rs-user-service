package error

import "errors"

var (
	ErrEmailAlreadyExist     = errors.New("email already exist")
	ErrUserAlreadyExist     = errors.New("user already exist")
	ErrUserNameExist     = errors.New("username already exist")
	ErrUserNotFound         = errors.New("user not found")
	ErrWrongPassword        = errors.New("wrong password")
	ErrPasswordDoesNotMatch = errors.New("password does not match")
	ErrInvalidToken = errors.New("invalid token")
	// ErrInvalidCredentials is returned for both "user not found" and "wrong
	// password" on login, so responses never reveal whether a username exists
	// (prevents user enumeration).
	ErrInvalidCredentials = errors.New("invalid username or password")
)

var UserErrors = []error{
	ErrEmailAlreadyExist,
	ErrUserAlreadyExist,
	ErrUserNameExist,
	ErrUserNotFound,
	ErrWrongPassword,
	ErrPasswordDoesNotMatch,
	ErrInvalidToken,
	ErrInvalidCredentials,
}
