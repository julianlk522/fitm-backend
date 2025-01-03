package handler

import (
	"log"
	"net/http"

	util "github.com/julianlk522/fitm/handler/util"

	"github.com/go-chi/render"

	"golang.org/x/crypto/bcrypt"

	"github.com/julianlk522/fitm/db"
	e "github.com/julianlk522/fitm/error"
	"github.com/julianlk522/fitm/model"
)

// Auth
func SignUp(w http.ResponseWriter, r *http.Request) {
	signup_data := &model.SignUpRequest{}
	if err := render.Bind(r, signup_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	if util.LoginNameTaken(signup_data.Auth.LoginName) {
		render.Render(w, r, e.ErrInvalidRequest(e.ErrLoginNameTaken))
		return
	}

	pw_hash, err := bcrypt.GenerateFromPassword(
		[]byte(signup_data.Auth.Password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Client.Exec(
		`INSERT INTO users VALUES (?,?,?,?,?,?)`,
		signup_data.ID,
		signup_data.Auth.LoginName,
		pw_hash,
		nil,
		nil,
		signup_data.CreatedAt,
	)
	if err != nil {
		log.Fatal(err)
	}

	token, err := util.GetJWTFromLoginName(signup_data.Auth.LoginName)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusCreated)
	util.RenderJWT(token, w, r)
}

func LogIn(w http.ResponseWriter, r *http.Request) {
	login_data := &model.LogInRequest{}
	if err := render.Bind(r, login_data); err != nil {
		render.Render(w, r, e.ErrInvalidRequest(err))
		return
	}

	is_authenticated, err := util.AuthenticateUser(login_data.LoginName, login_data.Password)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	} else if !is_authenticated {
		render.Render(w, r, e.ErrUnauthenticated(e.ErrInvalidLogin))
		return
	}

	token, err := util.GetJWTFromLoginName(login_data.Auth.LoginName)
	if err != nil {
		render.Render(w, r, e.Err500(err))
		return
	}

	render.Status(r, http.StatusOK)
	util.RenderJWT(token, w, r)
}
