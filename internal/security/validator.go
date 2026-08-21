package security

import (
	"errors"
	"net/mail"
	"regexp"
	"unicode"
)

var (
	ErrInvalidEmail       = errors.New("invalid email format")
	ErrPasswordTooShort   = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong    = errors.New("password must not exceed 128 characters")
	ErrPasswordUppercase  = errors.New("password must contain at least one uppercase letter")
	ErrPasswordLowercase  = errors.New("password must contain at least one lowercase letter")
	ErrPasswordNumber     = errors.New("password must contain at least one number")
	ErrPasswordSpecial    = errors.New("password must contain at least one special character")
	ErrNameTooShort       = errors.New("name must be at least 2 characters")
	ErrNameTooLong        = errors.New("name must not exceed 100 characters")
	ErrInvalidName        = errors.New("name contains invalid characters")
)

func ValidateEmail(email string) error {
	if email == "" {
		return ErrInvalidEmail
	}
	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	if len(password) > 128 {
		return ErrPasswordTooLong
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ErrPasswordUppercase
	}
	if !hasLower {
		return ErrPasswordLowercase
	}
	if !hasNumber {
		return ErrPasswordNumber
	}
	if !hasSpecial {
		return ErrPasswordSpecial
	}

	return nil
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z\s'-]{2,100}$`)

func ValidateName(name string) error {
	if len(name) < 2 {
		return ErrNameTooShort
	}
	if len(name) > 100 {
		return ErrNameTooLong
	}
	if !nameRegex.MatchString(name) {
		return ErrInvalidName
	}
	return nil
}
