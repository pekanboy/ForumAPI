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
	"strings"
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

	_, err := h.db.NamedExec(`INSERT INTO forum."user"(nickname, fullname, about, email) VALUES (:nickname, :fullname, :about, :email)`, &user)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var users []models.User
			err = h.db.Select(&users, `SELECT nickname, fullname, about, email FROM forum."user" WHERE nickname = $1 OR email = $2 LIMIT 2`, &user.Nickname, &user.Email)
			if err != nil {
				httputils.Respond(w, http.StatusInternalServerError, nil)
				return
			}
			httputils.Respond(w, http.StatusConflict, users)
			return
		}
	}
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusCreated, user)
}

func (h *Handlers) GetUser(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	nickname := params["nickname"]

	user := models.User{}

	err := h.db.Get(&user, `SELECT nickname, fullname, about, email FROM forum."user" WHERE nickname = $1`, nickname)

	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + nickname
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
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + nickname
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	err = h.db.QueryRowx(
		`UPDATE forum."user" 
			   SET fullname = COALESCE(NULLIF($1, ''), fullname),
			       about = COALESCE(NULLIF($2, ''), about),
			       email = COALESCE(NULLIF($3, ''), email) 
			   WHERE nickname = $4 
			   RETURNING nickname, fullname, about, email`,
		user.Fullname,
		user.About,
		user.Email,
		user.Nickname).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.About,
		&user.Email,
	)
	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			mes := models.Message{}
			mes.Message = "This email is already registered by user: " + nickname
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
	forum := models.Forum{}

	if err := json.NewDecoder(r.Body).Decode(&forum); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err := h.db.Get(&forum.User, `SELECT nickname FROM forum."user" WHERE nickname = $1`, forum.User)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find user with nickname: " + forum.User
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	_, err = h.db.NamedExec(
		`INSERT INTO forum.forum(title, "user", slug) 
			   VALUES (:title, :user, :slug)`,
		&forum)

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

	forum := models.Forum{}

	err := h.db.Get(&forum, `SELECT slug, title, "user", posts, threads FROM forum.forum WHERE slug = $1`, slug)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find forum with slug: " + slug
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

	thread := models.Thread{}

	if err := json.NewDecoder(r.Body).Decode(&thread); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err := h.db.Get(&thread.Author, `SELECT nickname FROM forum."user" WHERE nickname = $1`, thread.Author)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find thread author by nickname: " + thread.Author
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	err = h.db.Get(&thread.Forum, `SELECT slug FROM forum.forum WHERE slug = $1`, forum)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find thread forum by slug: " + thread.Forum
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if thread.Created.IsZero() {
		thread.Created = time.Now()
	}

	err = h.db.QueryRowx(`
		INSERT INTO forum.thread(title, author, forum, message, votes, slug, created) 
		VALUES ($1, $2, $3, $4, $5, nullif($6, ''), $7)
		RETURNING id`,
		thread.Title,
		thread.Author,
		thread.Forum,
		thread.Message,
		thread.Votes,
		thread.Slug,
		thread.Created).Scan(
		&thread.Id,
	)

	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			var result models.Thread
			err := h.db.Get(&result, `
		SELECT id, title, author, forum, message, votes, slug, created 
		FROM forum.thread 
		WHERE slug = $1`,
				thread.Slug)
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
		mes := models.Message{}
		mes.Message = "Can't find forum by slug: " + forum
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
	if since == "" {
		if desc {
			err = h.db.Select(&users,
				  `select DISTINCT u.nickname, u.fullname, u.about, u.email 
						from forum.user u
								 left join forum.thread t on t.author = u.nickname and t.forum = $1
								 left join
							 (select p.author
							  from forum.post p
							  where p.forum = $1) as p
							 on p.author = u.nickname
						where (t.id is not null or p.author is not null)
						order by u.nickname desc 
					limit $2`,
				&forum,
				&limit)
		} else {
			err = h.db.Select(&users,
				  `select DISTINCT u.nickname, u.fullname, u.about, u.email 
						from forum.user u
								 left join forum.thread t on t.author = u.nickname and t.forum = $1
								 left join
							 (select p.author
							  from forum.post p
							  where p.forum = $1) as p
							 on p.author = u.nickname
						where (t.id is not null or p.author is not null)
						order by u.nickname 
					    limit $2`,
				&forum,
				&limit)
		}
	} else {
		if desc {
			err = h.db.Select(&users,
				`select DISTINCT u.nickname, u.fullname, u.about, u.email 
						from forum.user u
								 left join forum.thread t on t.author = u.nickname and t.forum = $1
								 left join
							 (select p.author
							  from forum.post p
							  where p.forum = $1) as p
							 on p.author = u.nickname
						where (t.id is not null or p.author is not null) and u.nickname < $3
						order by u.nickname desc 
					limit $2`,
				&forum,
				&limit,
				&since)
		} else {
			err = h.db.Select(&users,
				`select DISTINCT u.nickname, u.fullname, u.about, u.email 
						from forum.user u
								 left join forum.thread t on t.author = u.nickname and t.forum = $1
								 left join
							 (select p.author
							  from forum.post p
							  where p.forum = $1) as p
							 on p.author = u.nickname
						where (t.id is not null or p.author is not null) and u.nickname > $3
						order by u.nickname asc 
					limit $2`,
				&forum,
				&limit,
				&since)
		}
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if users == nil {
		httputils.Respond(w, http.StatusOK, []models.User{})
	} else {
		httputils.Respond(w, http.StatusOK, users)
	}
}

func (h *Handlers) GetForumThreads(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	forum := params["slug"]

	var contained string
	err := h.db.Get(&contained, `SELECT slug FROM forum.forum WHERE slug = $1`, forum)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find forum by slug: " + forum
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

	var threads []models.Thread
	if since == "" {
		if desc {
			err = h.db.Select(&threads, `
						select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created 
						from forum.thread t 
						where t.forum = $1
						order by t.created desc 
						limit $2`,
				&forum,
				&limit)
		} else {
			err = h.db.Select(&threads, `
						select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created 
						from forum.thread t 
						where t.forum = $1
						order by t.created 
						limit $2`,
				&forum,
				&limit)
		}
	} else {
		if desc {
			err = h.db.Select(&threads, `
						select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created 
						from forum.thread t 
						where t.forum = $1 and t.created <= $3 
						order by t.created desc 
						limit $2`,
				&forum,
				&limit,
				&since)
		} else {
			err = h.db.Select(&threads, `
						select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created 
						from forum.thread t 
						where t.forum = $1 and t.created >= $3 
						order by t.created 
						limit $2`,
				&forum,
				&limit,
				&since)
		}
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if threads != nil {
		httputils.Respond(w, http.StatusOK, threads)
	} else {
		httputils.Respond(w, http.StatusOK, []models.Thread{})
	}
}

// POST

func (h *Handlers) GetPost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	post := params["id"]

	var related []string
	related = strings.Split(r.URL.Query().Get("related"), ",")

	var result struct {
		Post   *models.Post   `json:"post,omitempty"`
		Thread *models.Thread `json:"thread,omitempty"`
		Forum  *models.Forum  `json:"forum,omitempty"`
		User   *models.User   `json:"author,omitempty"`
	}

	var p models.Post
	err := h.db.Get(&p, `SELECT id, parent, author, message, isEdited, forum, thread, created FROM forum.post WHERE id = $1 LIMIT 1`, post)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find post with id: " + post
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	result.Post = &p

	var user models.User
	var forum models.Forum
	var thread models.Thread

	for _, item := range related {
		if item == "user" {
			err = h.db.Get(&user, `SELECT nickname, fullname, about, email FROM forum.user WHERE nickname = $1`, result.Post.Author)
			result.User = &user
		}
		if item == "forum" {
			err = h.db.Get(&forum, `SELECT title, "user", slug, posts, threads FROM forum.forum WHERE slug = $1`, result.Post.Forum)
			result.Forum = &forum
		}
		if item == "thread" {
			err = h.db.Get(&thread, `SELECT id, title, author, forum, message, votes, coalesce(slug, '') as slug, created FROM forum.thread WHERE id = $1`, result.Post.Thread)
			result.Thread = &thread
		}
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
	}

	httputils.Respond(w, http.StatusOK, result)
}

func (h *Handlers) ChangePost(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	id, err := strconv.Atoi(params["id"])
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	post := models.Post{Id: id}

	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	tx, err := h.db.Beginx()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err = tx.QueryRowx(`
UPDATE forum.post 
SET message = COALESCE(nullif($1, ''), message), isEdited = CASE $1 WHEN message THEN false WHEN '' THEN false ELSE true end
WHERE id = $2 
RETURNING id, parent, author, message, isEdited, forum, thread, created `,
		post.Message,
		post.Id).Scan(
		&post.Id,
		&post.Parent,
		&post.Author,
		&post.Message,
		&post.IsEdited,
		&post.Forum,
		&post.Thread,
		&post.Created,
	)

	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find post with id: " + strconv.Itoa(id)
		httputils.Respond(w, http.StatusNotFound, mes)
		_ = tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	httputils.Respond(w, http.StatusOK, post)
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

	var mes models.Message

	var info models.Thread
	if isId == -1 {
		err = h.db.Get(&info, `SELECT id, forum FROM forum.thread WHERE slug = $1`, thread)
		if errors.Is(err, sql.ErrNoRows) {
			mes.Message = "Can't find post thread by slug: " + thread
		}
	} else {
		err = h.db.Get(&info, `SELECT id, forum FROM forum.thread WHERE id = $1`, isId)
		if errors.Is(err, sql.ErrNoRows) {
			mes.Message = "Can't find post thread by id: " + strconv.Itoa(isId)
		}
	}

	if errors.Is(err, sql.ErrNoRows) {
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	tx, _ := h.db.Beginx()
	create := time.Now()

	for index, item := range posts {
		var contained string
		err := tx.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, item.Author)
		if errors.Is(err, sql.ErrNoRows) {
			mes := models.Message{}
			mes.Message = "Can't find post author by nickname: " + item.Author
			httputils.Respond(w, http.StatusNotFound, mes)
			_ = tx.Rollback()
			return
		}

		item.Thread = info.Id

		if item.Parent != 0 {
			var parentExiste string
			err = tx.Get(&parentExiste, `SELECT id FROM forum.post WHERE id = $1 and thread = $2`, item.Parent, item.Thread)

			if err != nil {
				mes := models.Message{}
				mes.Message = "Parent post was created in another thread"
				httputils.Respond(w, http.StatusConflict, mes)
				_ = tx.Rollback()
				return
			}
		}

		item.Forum = info.Forum
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
		mes := models.Message{}
		mes.Message = "Can't find thread by slug or id: " + thread
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	httputils.Respond(w, http.StatusOK, result)
}

func (h *Handlers) ChangeThread(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	thread := params["slug_or_id"]

	isId, err := strconv.Atoi(thread)
	if err != nil {
		isId = -1
	}

	result := models.Thread{Slug: thread, Id: isId}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var mes models.Message
	tx, err := h.db.Beginx()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if isId == -1 {
		err = tx.QueryRowx(`UPDATE forum.thread SET title = COALESCE(nullif($1, ''), title), message = COALESCE(nullif($2, ''), message) WHERE slug = $3 RETURNING *`,
			result.Title,
			result.Message,
			result.Slug).Scan(
			&result.Id,
			&result.Title,
			&result.Author,
			&result.Forum,
			&result.Message,
			&result.Votes,
			&result.Slug,
			&result.Created)
		mes.Message = "Can't find thread by slug: " + thread
	} else {
		err = tx.QueryRowx(`UPDATE forum.thread SET title = COALESCE(nullif($1, ''), title), message = COALESCE(nullif($2, ''), message) WHERE id = $3 RETURNING *`,
			result.Title,
			result.Message,
			result.Id).Scan(
			&result.Id,
			&result.Title,
			&result.Author,
			&result.Forum,
			&result.Message,
			&result.Votes,
			&result.Slug,
			&result.Created)
		mes.Message = "Can't find forum by id: " + thread
	}

	if err != nil {
		httputils.Respond(w, http.StatusNotFound, mes)
		_ = tx.Rollback()
		return
	}

	err = tx.Commit()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	httputils.Respond(w, http.StatusOK, result)
}

func (h *Handlers) CreateVote(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	thread := params["slug_or_id"]

	isId, err := strconv.Atoi(thread)
	if err != nil {
		isId = -1
	}

	var vote models.Vote

	if err := json.NewDecoder(r.Body).Decode(&vote); err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var contained string
	err = h.db.Get(&contained, `SELECT nickname FROM forum."user" WHERE nickname = $1`, vote.Nickname)
	if errors.Is(err, sql.ErrNoRows) {
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + vote.Nickname
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if isId == -1 {
		err = h.db.Get(&vote.Thread, `SELECT id as thread FROM forum.thread WHERE slug = $1`, thread)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by slug: " + thread
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	} else {
		vote.Thread = isId
	}

	var vot int
	err = h.db.Get(&vot, `SELECT voice FROM forum.vote WHERE thread = $1 and nickname = $2`, vote.Thread, vote.Nickname)
	if errors.Is(err, sql.ErrNoRows) {
		_, err = h.db.NamedExec(`INSERT INTO forum.vote(thread, nickname, voice) VALUES (:thread, :nickname, :voice)`, &vote)
	} else {
		if vot != vote.Voice {
			_, err = h.db.NamedExec(`UPDATE forum.vote SET voice = :voice WHERE thread = :thread and nickname = :nickname`, &vote)
		}
	}

	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find thread by id: " + thread
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	var result models.Thread
	if isId == -1 {
		err = h.db.Get(&result, `SELECT id, title, author, forum, message, votes, slug, created FROM forum.thread WHERE slug = $1`, thread)
	} else {
		err = h.db.Get(&result, `SELECT id, title, author, forum, message, votes, slug, created FROM forum.thread WHERE id = $1`, isId)
	}

	httputils.Respond(w, http.StatusOK, result)
}

func (h *Handlers) ThreadPosts(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	thread := params["slug_or_id"]

	isId, err := strconv.Atoi(thread)
	if err != nil {
		isId = -1
	}

	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 100
	}

	since, err := strconv.Atoi(r.URL.Query().Get("since"))
	if err != nil {
		since = 0
	}

	sort := r.URL.Query().Get("sort")

	desc, err := strconv.ParseBool(r.URL.Query().Get("desc"))
	if err != nil {
		desc = false
	}

	var id int
	if isId != -1 {
		id = isId
		rows, err := h.db.Query(`SELECT id as thread FROM forum.thread WHERE id = $1 LIMIT 1`, isId)
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		if !rows.Next() {
			mes := models.Message{}
			mes.Message = "Can't find thread by id: " + thread
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}

		_ = rows.Close()
	} else {
		err = h.db.Get(&id, `SELECT id as thread FROM forum.thread WHERE slug = $1 LIMIT 1`, thread)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by slug: " + thread
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	}

	var posts []models.Post

	// Float
	switch sort {
	case "tree":
		if since == 0 {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE thread = $1 
							ORDER BY path DESC, id DESC 
							LIMIT $2`,
							id, limit)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE thread = $1 
							ORDER BY path, id 
							LIMIT $2`,
					id, limit)
			}
		} else {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE thread = $1 and path < (SELECT path FROM forum.post WHERE id = $3 LIMIT 1)
							ORDER BY path DESC, id DESC 
							LIMIT $2`,
					id, limit, since)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE thread = $1 and path > (SELECT path FROM forum.post WHERE id = $3 LIMIT 1)
							ORDER BY path, id 
							LIMIT $2`,
					id, limit, since)
			}
		}
	case "parent_tree":
		if since == 0 {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE path[1] IN (
								SELECT id 
								FROM forum.post 
								WHERE thread = $1 and parent = 0
								ORDER BY id DESC 
								LIMIT $2)
							ORDER BY path[1] DESC, path, id`,
							id, limit)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE path[1] IN (
								SELECT id 
								FROM forum.post 
								WHERE thread = $1 AND parent = 0
								ORDER BY id
								LIMIT $2)
							ORDER BY path`,
							id, limit)
			}
		} else {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE path[1] IN (
								SELECT id 
								FROM forum.post 
								WHERE thread = $1 AND parent = 0 and path[1] < (SELECT path[1] FROM forum.post WHERE id = $3 LIMIT 1)
								ORDER BY id DESC 
								LIMIT $2)
							ORDER BY path[1] DESC, path, id`,
							id, limit, since)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread 
							FROM forum.post
							WHERE path[1] in (
								SELECT id 
								FROM forum.post 
								WHERE thread = $1 AND parent = 0 and path[1] > (SELECT path[1] FROM forum.post WHERE id = $3 LIMIT 1)
								ORDER BY id ASC
								LIMIT $2)
							ORDER BY path, id`,
					id, limit, since)
			}
		}
	default:
		if since == 0 {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread
					   FROM forum.post
					   WHERE thread = $1
					   ORDER BY id DESC
					   LIMIT $2`,
					id,
					limit,
				)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread
					   FROM forum.post
					   WHERE thread = $1
					   ORDER BY id
					   LIMIT $2`,
					id,
					limit,
				)
			}
		} else {
			if desc {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread
					   FROM forum.post
					   WHERE thread = $1 and id < $3
					   ORDER BY id DESC
					   LIMIT $2`,
					id,
					limit,
					since,
				)
			} else {
				err = h.db.Select(&posts,
					`SELECT id, author, created, forum, isEdited, message, parent, thread
					   FROM forum.post
					   WHERE thread = $1 and id > $3
					   ORDER BY id
					   LIMIT $2`,
					id,
					limit,
					since,
				)
			}
		}
	}

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if posts == nil {
		httputils.Respond(w, http.StatusOK, []models.Post{})
		return
	}
	httputils.Respond(w, http.StatusOK, posts)

}

// SERVICE

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

func (h *Handlers) AllInfo(w http.ResponseWriter, r *http.Request) {
	var status models.Status

	err := h.db.QueryRow(`SELECT COUNT(*) FROM forum."user";`).Scan(&status.User)
	if err != nil {
		status.User = 0
	}
	err = h.db.QueryRow(`SELECT COUNT(*) FROM forum.forum;`).Scan(&status.Forum)
	if err != nil {
		status.Forum = 0
	}
	err = h.db.QueryRow(`SELECT COUNT(*) FROM forum.thread;`).Scan(&status.Thread)
	if err != nil {
		status.Thread = 0
	}
	err = h.db.QueryRow(`SELECT COUNT(*) FROM forum.post;`).Scan(&status.Post)
	if err != nil {
		status.Post = 0
	}

	httputils.Respond(w, http.StatusOK, status)
}
