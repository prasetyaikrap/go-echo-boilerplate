package security

import (
	"errors"
	"go-serviceboilerplate/commons/utils"
	"go-serviceboilerplate/infrastrucutres/configurations"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

type PasswordHashSecurity struct {
	Configs *configurations.Configs
	Cost    int
}

func NewPasswordHashSecurity(configs *configurations.Configs, cost int) *PasswordHashSecurity {
	return &PasswordHashSecurity{Cost: cost, Configs: configs}
}

func (s *PasswordHashSecurity) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), s.Cost)
	if(err != nil) {
		return "", utils.NewInvariantError(err)
	}
    return string(bytes), nil
}

func (s *PasswordHashSecurity) ComparePassword(password, passwordHash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	return err == nil 
}

func (s *PasswordHashSecurity) ValidatePassword(password string) error {
	if len(password) < 8 {
		return utils.NewInvariantError(errors.New("password must be at least 8 characters long"))
	}

	var hasLetter, hasDigit, hasSpecial, hasSpace, hasUnsafe bool

	for _, ch := range password {
		switch {
		case unicode.IsLetter(ch):
			hasLetter = true
		case unicode.IsDigit(ch):
			hasDigit = true
		case unicode.IsSpace(ch):
			hasSpace = true
		case utils.IsSafeSpecialChar(ch):
			hasSpecial = true
		case unicode.IsPunct(ch) || unicode.IsSymbol(ch):
			hasUnsafe = true
		}
	}

	if hasSpace {
		return utils.NewInvariantError(errors.New("password cannot contain spaces"))
	}
	if hasUnsafe {
		return utils.NewInvariantError(errors.New("password contains unsafe special characters"))
	}
	if !hasLetter {
		return utils.NewInvariantError(errors.New("password must contain at least one letter"))
	}
	if !hasDigit {
		return utils.NewInvariantError(errors.New("password must contain at least one digit"))
	}
	if !hasSpecial {
		return utils.NewInvariantError(errors.New("password must contain at least one special character"))
	}

	return nil
}