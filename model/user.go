package model

import (
	"net/http"

	e "github.com/julianlk522/fitm/error"
	util "github.com/julianlk522/fitm/model/util"

	"github.com/google/uuid"
)

// AUTH
type Auth struct {
	LoginName string `json:"login_name"`
	Password  string `json:"password"`
}
type SignUpRequest struct {
	*Auth
	ID        string
	CreatedAt string
}

func (sr *SignUpRequest) Bind(r *http.Request) error {
	switch {
	case sr.Auth.LoginName == "":
		return e.ErrNoLoginName
	case len(sr.Auth.LoginName) < util.LOGIN_NAME_LOWER_LIMIT:
		return e.LoginNameExceedsLowerLimit(util.LOGIN_NAME_LOWER_LIMIT)
	case len(sr.Auth.LoginName) > util.LOGIN_NAME_UPPER_LIMIT:
		return e.LoginNameExceedsUpperLimit(util.LOGIN_NAME_UPPER_LIMIT)
	case util.ContainsInvalidChars(sr.Auth.LoginName):
		return e.ErrLoginNameContainsInvalidChars

	case sr.Auth.Password == "":
		return e.ErrNoPassword
	case len(sr.Auth.Password) < util.PASSWORD_LOWER_LIMIT:
		return e.PasswordExceedsLowerLimit(util.PASSWORD_LOWER_LIMIT)
	case len(sr.Auth.Password) > util.PASSWORD_UPPER_LIMIT:
		return e.PasswordExceedsUpperLimit(util.PASSWORD_UPPER_LIMIT)
	}

	sr.ID = uuid.New().String()
	sr.CreatedAt = util.NEW_SHORT_TIMESTAMP()
	return nil
}

type LogInRequest struct {
	*Auth
}

func (lr *LogInRequest) Bind(r *http.Request) error {
	if lr.Auth.LoginName == "" {
		return e.ErrNoLoginName
	} else if lr.Auth.Password == "" {
		return e.ErrNoPassword
	}

	return nil
}