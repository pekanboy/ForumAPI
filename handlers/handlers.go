package handlers

import (
	"database/sql"
	"encoding/json"
	"errors"
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

// USER

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{Nickname: nickname}

	//limit := r.URL.Query().Get("limit")

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	_, err = tx.NamedExec(`INSERT INTO forum."user"(nickname, fullname, about, email) VALUES (:nickname, :fullname, :about, :email)`, &user)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var users []models.User
			err := tx.Select(&users, `SELECT nickname, fullname, about, email FROM forum."user" WHERE nickname = $1 OR email = $2`, user.Nickname, user.Email)
			if err != nil {
				httputils.Respond(w, http.StatusInternalServerError, nil)
				_ = tx.Rollback()
				return
			}
			httputils.Respond(w, http.StatusConflict, users)
			return
		}
	}
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	if err = tx.Commit(); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusCreated, user)
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request)  {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{Nickname: nickname}

	err := h.db.Get(&user,`SELECT fullname, about, email FROM forum."user" WHERE nickname = $1`, nickname)

	if errors.Is(err, sql.ErrNoRows) {
		mes := 	"Can't find user by nickname: " + nickname
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusOK, user)
}

func (h *Handlers) ChangeUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{Nickname: nickname}

	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var contained string
	err := h.db.Get(&contained,`SELECT nickname FROM forum."user" WHERE nickname = $1`, nickname)
	if errors.Is(err, sql.ErrNoRows) {
		mes := 	"Can't find user by nickname: " + nickname
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	_, err = h.db.NamedExec(`UPDATE forum."user" SET fullname = :fullname, about = :about, email = :email WHERE nickname = :nickname`, &user)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			mes := "This email is already registered by user: " + nickname
			httputils.Respond(w, http.StatusConflict, mes)
			return
		}
	}
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusOK, user)
}

// FORUM

func (h *Handlers) CreateForum(w http.ResponseWriter, r *http.Request) {
	forum := models.Forum{Posts: 0, Threads: 0}

	if err := json.NewDecoder(r.Body).Decode(&forum); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var contained string
	err := h.db.Get(&contained,`SELECT nickname FROM forum."user" WHERE nickname = $1`, forum.User)
	if errors.Is(err, sql.ErrNoRows) {
		mes := 	"Can't find user with nickname: " + forum.User
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	_, err = h.db.NamedExec(`INSERT INTO forum.forum(title, "user", slug, posts, threads) VALUES (:title, :user, :slug, :posts, :threads)`, &forum)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var result models.Forum
			err := h.db.Get(&result, `SELECT title, "user", slug, posts, threads FROM forum.forum WHERE slug = $1`, forum.Slug)
			if err != nil {
				httputils.Respond(w, http.StatusInternalServerError, nil)
				return
			}
			httputils.Respond(w, http.StatusConflict, result)
			return
		}
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusCreated, forum)
}