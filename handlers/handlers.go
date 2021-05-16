package handlers

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"
	"net/http"
	"server/httputils"
	"server/models"
)

type Handlers struct {
	db *sqlx.DB
}

func NewHandler(db *sqlx.DB) *Handlers {
	return &Handlers{
		db: db,
	}
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{Nickname: nickname}

	//limit := r.URL.Query().Get("limit")

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	tx, _:= h.db.Beginx()
	_, err := tx.NamedExec(`INSERT INTO user_test(nickname, fullname, about, email) VALUES (:nickname, :fullname, :about, :email)`, &user)
	_ = tx.Commit()
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var users []models.User
			err := h.db.Select(&users, "SELECT nickname, fullname, about, email FROM user_test WHERE nickname = $1 OR email = $2", user.Nickname, user.Email)
			if err != nil {
				httputils.Respond(w, http.StatusInternalServerError, nil)
				return
			}
			httputils.Respond(w, http.StatusConflict, users)
			return
		}
	}
	httputils.Respond(w, http.StatusCreated, user)
}
