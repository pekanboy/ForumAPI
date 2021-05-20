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
	"strconv"
	"time"
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

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{Nickname: nickname}

	err := h.db.Get(&user, `SELECT fullname, about, email FROM forum."user" WHERE nickname = $1`, nickname)

	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find user by nickname: " + nickname
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
	err := h.db.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, nickname)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find user by nickname: " + nickname
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
	err := h.db.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, forum.User)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find user with nickname: " + forum.User
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

func (h *Handlers) GetForum(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]

	forum := models.Forum{Slug: slug}

	err := h.db.Get(&forum, `SELECT title, "user", posts, threads FROM forum.forum WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find forum with slug: " + slug
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusOK, forum)
}

func (h *Handlers) CreateThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	forum := params["slug"]

	thread := models.Thread{Forum: forum, Votes: 0}

	if err := json.NewDecoder(r.Body).Decode(&thread); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var contained string
	err := h.db.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, thread.Author)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find thread author by nickname: " + thread.Author
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	err = h.db.Get(&contained, `SELECT slug FROM forum.forum WHERE slug = $1`, thread.Forum)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find thread forum by slug: " + thread.Forum
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if thread.Created.IsZero() {
		thread.Created = time.Now()
	}

	_, err = h.db.NamedExec(`INSERT INTO forum.thread(title, author, forum, message, votes, slug, created) VALUES (:title, :author, :forum, :message, :votes, :slug, :created)`, &thread)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var result models.Thread
			err := h.db.Get(&result, `SELECT id, title, author, forum, message, votes, slug, created FROM forum.thread WHERE slug = $1`, thread.Slug)
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

	httputils.Respond(w, http.StatusCreated, thread)
}

func (h *Handlers) GetForumUsers(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	forum := params["slug"]

	var contained string
	err := h.db.Get(&contained, `SELECT slug FROM forum.forum WHERE slug = $1`, forum)
	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find forum by slug: " + forum
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 100
	}

	since := r.URL.Query().Get("since")
	desc, err := strconv.ParseBool(r.URL.Query().Get("desc"))
	if err != nil {
		desc = false
	}

	var users []models.User
	if desc {
		err = h.db.Get(&users, `select u.nickname, u.fullname, u.about, u.email from forum.user u inner join forum.thread t on t.author = u.nickname and t.forum = $1 where u.nickname in (select p.author from forum.post p where p.forum = $1 and p.author = u.nickname and u.nickname > $3) order by u.nickname desc limit $2`, &forum, &limit, &since)
	} else {
		err = h.db.Get(&users, `select u.nickname, u.fullname, u.about, u.email from forum.user u inner join forum.thread t on t.author = u.nickname and t.forum = $1 where u.nickname in (select p.author from forum.post p where p.forum = $1 and p.author = u.nickname and u.nickname > $3) order by u.nickname asc limit $2`, &forum, &limit, &since)
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusOK, users)
}

// THREAD

func (h *Handlers) CreatePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	thread := params["slug_or_id"]

	isId, err := strconv.Atoi(thread)
	if err != nil {
		isId = -1
	}

	var posts []models.Post

	if err := json.NewDecoder(r.Body).Decode(&posts); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	tx, _ := h.db.Beginx()
	create := time.Now()

	for index, item := range posts {
		var contained string
		err := tx.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, item.Author)
		if errors.Is(err, sql.ErrNoRows) {
			mes := "Can't find post author by nickname: " + item.Author
			httputils.Respond(w, http.StatusNotFound, mes)
			_ = tx.Rollback()
			return
		}

		if item.Parent != 0 {
			var parentExiste string
			err = tx.Get(&parentExiste, `SELECT id FROM forum.post WHERE id = $1`, item.Parent)
			if err != nil {
				mes := "Parent post was created in another thread"
				httputils.Respond(w, http.StatusConflict, mes)
				_ = tx.Rollback()
				return
			}
		}

		var mes string

		if isId == -1 {
			err = tx.Get(&item, `SELECT id, forum FROM forum.thread WHERE slug = $1`, thread)
			if errors.Is(err, sql.ErrNoRows) {
				mes = "Can't find post thread by slug: " + thread
			}
		} else {
			err = tx.Get(&item, `SELECT id, forum FROM forum.thread WHERE id = $1`, isId)
			if errors.Is(err, sql.ErrNoRows) {
				mes = "Can't find post thread by id: " + strconv.Itoa(isId)
			}
		}

		if errors.Is(err, sql.ErrNoRows) {
			httputils.Respond(w, http.StatusNotFound, mes)
			_ = tx.Rollback()
			return
		}

		item.Thread = item.Id
		item.Created = create
		item.IsEdited = false

		err = tx.QueryRowx(`INSERT INTO forum.post(parent, author, message, isEdited, forum, thread, created) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
			item.Parent,
			item.Author,
			item.Message,
			item.IsEdited,
			item.Forum,
			item.Thread,
			item.Created).Scan(&item.Id)
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			_ = tx.Rollback()
			return
		}

		posts[index] = item
	}

	err = tx.Commit()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	httputils.Respond(w, http.StatusCreated, posts)
}

func (h *Handlers) GetThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	thread := params["slug_or_id"]

	isId, err := strconv.Atoi(thread)
	if err != nil {
		isId = -1
	}

	var result models.Thread
	if isId == -1 {
		err = h.db.Get(&result, `SELECT id, title, author, forum, message, votes, slug, created FROM forum.thread WHERE slug = $1`, thread)
	} else {
		err = h.db.Get(&result, `SELECT id, title, author, forum, message, votes, slug, created FROM forum.thread WHERE id = $1`, isId)
	}

	if errors.Is(err, sql.ErrNoRows) {
		mes := "Can't find thread by slug or id: " + thread
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	httputils.Respond(w, http.StatusOK, result)
}

func (h *Handlers) ChangeThread(w http.ResponseWriter, r *http.Request) {
	// Todo Спросить у Олега поля приходят только те. которые надо изменить?
}

// SERVICE

//  Todo No testing
func (h *Handlers) AllClear(w http.ResponseWriter, r *http.Request) {
	var err error
	tx, err := h.db.Beginx()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}
	_, err = tx.Exec(`TRUNCATE forum.forum CASCADE;`)
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec(`TRUNCATE forum.post CASCADE;`)
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec(`TRUNCATE forum.thread CASCADE;`)
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec(`TRUNCATE forum."user" CASCADE;`)
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	//_, err = tx.Exec(`TRUNCATE votes CASCADE;`) // TODO add

	err = tx.Commit()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	httputils.Respond(w, http.StatusOK, nil)
}

