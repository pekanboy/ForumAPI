package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/go-openapi/strfmt"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx"
	"net/http"
	"server/httputils"
	"server/models"
	"strconv"
	"strings"
	"time"
)

type Handlers struct {
	conn *pgx.ConnPool
}

func NewHandler(conn *pgx.ConnPool) *Handlers {
	return &Handlers{
		conn: conn,
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

	_, err := h.conn.Exec("insertUser",
		user.Nickname,
		user.Fullname,
		user.About,
		user.Email)

	if driverErr, ok := err.(pgx.PgError); ok {
		if driverErr.Code == "23505" {
			row, err := h.conn.Query("selectDublicateUser", user.Nickname, user.Email)
			if err != nil {
				httputils.Respond(w, http.StatusInternalServerError, nil)
				return
			}
			defer row.Close()
			var users []models.User
			for row.Next() {
				user := models.User{}
				err = row.Scan(
					&user.Nickname,
					&user.Fullname,
					&user.About,
					&user.Email)
				if err != nil {
					httputils.Respond(w, http.StatusInternalServerError, nil)
					return
				}

				users = append(users, user)
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

	row, _ := h.conn.Query("selectUser", nickname)

	if !row.Next() {
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + nickname
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	defer row.Close()

	err := row.Scan(&user.Nickname, &user.Fullname, &user.About, &user.Email)
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	row, _ := tx.Query("checkUser", nickname)

	if !row.Next() {
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + nickname
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}
	row.Close()

	err = tx.QueryRow(
		"changeUser",
		user.Fullname,
		user.About,
		user.Email,
		user.Nickname).Scan(
		&user.Nickname,
		&user.Fullname,
		&user.About,
		&user.Email,
	)
	if err != nil {
		mes := models.Message{}
		mes.Message = "This email is already registered by user: " + nickname
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusConflict, mes)
		return
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err = tx.QueryRow("checkUser", forum.User).Scan(&forum.User)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find user with nickname: " + forum.User
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	_, err = tx.Exec(
		"insertForum",
		&forum.Title, &forum.User, &forum.Slug)

	if err != nil {
		_ = tx.Rollback()
		tx, err = h.conn.Begin()
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		var result models.Forum
		err = tx.QueryRow("selectForum", forum.Slug).Scan(
			&result.Title, &result.User, &result.Slug, &result.Posts, &result.Threads)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		httputils.Respond(w, http.StatusConflict, result)
		return

	}

	err = tx.Commit()
	httputils.Respond(w, http.StatusCreated, forum)
}

func (h *Handlers) GetForum(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	slug := params["slug"]

	forum := models.Forum{}

	err := h.conn.QueryRow("selectForum", slug).Scan(
		&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find forum with slug: " + slug
		httputils.Respond(w, http.StatusNotFound, mes)
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err = tx.QueryRow("checkUser", thread.Author).Scan(&thread.Author)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find thread author by nickname: " + thread.Author
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	err = tx.QueryRow("checkForum", forum).Scan(&thread.Forum)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find thread forum by slug: " + thread.Forum
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}
	if thread.Created.String() == "" {
		thread.Created = time.Now()
	}

	err = tx.QueryRow("insertThread",
		thread.Title,
		thread.Author,
		thread.Forum,
		thread.Message,
		thread.Votes,
		thread.Slug,
		thread.Created).Scan(&thread.Id)

	if err != nil {
		_ = tx.Rollback()
		tx, err = h.conn.Begin()
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		var result models.Thread
		err := tx.QueryRow("selectThread",
			thread.Slug).Scan(
			&result.Id,
			&result.Title,
			&result.Author,
			&result.Forum,
			&result.Message,
			&result.Votes,
			&result.Slug,
			&result.Created)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		httputils.Respond(w, http.StatusConflict, result)
		return
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusCreated, thread)
}

func (h *Handlers) GetForumUsers(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	forum := params["slug"]

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	row, _ := tx.Query("checkForum", forum)
	if !row.Next() {
		mes := models.Message{}
		mes.Message = "Can't find forum by slug: " + forum
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	row.Close()

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
			row, err = tx.Query(
				"selectUserOrderDesc",
				&forum,
				&limit)
		} else {
			row, err = tx.Query(
				"selectUserOrder",
				&forum,
				&limit)
		}
	} else {
		if desc {
			row, err = tx.Query(
				"selectUserWhereOrderDesc",
				&forum,
				&limit,
				&since)
		} else {
			row, err = tx.Query(
				"selectUserWhereOrder",
				&forum,
				&limit,
				&since)
		}
	}

	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	for row.Next() {
		u := models.User{}
		err = row.Scan(
			&u.Nickname,
			&u.Fullname,
			&u.About,
			&u.Email,
		)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
		users = append(users, u)
	}

	if users == nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusOK, []models.User{})
	} else {
		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
		httputils.Respond(w, http.StatusOK, users)
	}
}

func (h *Handlers) GetForumThreads(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	forum := params["slug"]

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	row, _ := tx.Query("checkForum", forum)
	if !row.Next() {
		mes := models.Message{}
		mes.Message = "Can't find forum by slug: " + forum
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	row.Close()

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
			row, err = tx.Query("selectThreadOrderDesc",
				&forum,
				&limit)
		} else {
			row, err = tx.Query("selectThreadOrder",
				&forum,
				&limit)
		}
	} else {
		if desc {
			row, err = tx.Query("selectThreadWhereOrderDesc",
				&forum,
				&limit,
				&since)
		} else {
			row, err = tx.Query("selectThreadWhereOrder",
				&forum,
				&limit,
				&since)
		}
	}

	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	for row.Next() {
		t := models.Thread{}
		err = row.Scan(
			&t.Id,
			&t.Title,
			&t.Author,
			&t.Forum,
			&t.Message,
			&t.Votes,
			&t.Slug,
			&t.Created)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
		threads = append(threads, t)
	}

	row.Close()

	if threads != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusOK, threads)
	} else {
		err = tx.Commit()
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var p models.Post
	err = tx.QueryRow( "selectPost", post).Scan(
		&p.Id, &p.Parent, &p.Author, &p.Message, &p.IsEdited, &p.Forum, &p.Thread, &p.Created)
	if err != nil {
		mes := models.Message{}
		mes.Message = "Can't find post with id: " + post
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	result.Post = &p

	var user models.User
	var forum models.Forum
	var thread models.Thread

	for _, item := range related {
		if item == "user" {
			err = tx.QueryRow( "selectUser", result.Post.Author).Scan(
				&user.Nickname, &user.Fullname, &user.About, &user.Email)
			result.User = &user
		}
		if item == "forum" {
			err = tx.QueryRow( "selectForum", result.Post.Forum).Scan(
				&forum.Title, &forum.User, &forum.Slug, &forum.Posts, &forum.Threads)
			result.Forum = &forum
		}
		if item == "thread" {
			err = tx.QueryRow( "selectThreadById", result.Post.Thread).Scan(
				&thread.Id, &thread.Title, &thread.Author, &thread.Forum, &thread.Message, &thread.Votes, &thread.Slug, &thread.Created)
			result.Thread = &thread
		}
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	err = tx.QueryRow("updatePost",
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if isId == -1 {
		err = tx.QueryRow("selectIdForumThreadBySlug", thread).Scan(&info.Id, &info.Forum)
		if err != nil {
			mes.Message = "Can't find post thread by slug: " + thread
		}
	} else {
		err = tx.QueryRow("selectIdForumThreadById", isId).Scan(&info.Id, &info.Forum)
		if err != nil {
			mes.Message = "Can't find post thread by id: " + strconv.Itoa(isId)
		}
	}

	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	if len(posts) == 0 {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusCreated, posts)
		return
	}

	create := strfmt.DateTime(time.Now())

	var values string
	var args []interface{}
	l := len(posts) - 1

	if posts[0].Parent != 0 {
		var parent int

		row, _ := tx.Query("selectThreadIdFromPost", posts[0].Parent)

		if row.Next() {
			err := row.Scan(&parent)
			if err != nil {
				_ = tx.Rollback()
				return
			}
		}

		row.Close()

		if parent != info.Id {
			mes := models.Message{}
			mes.Message = "Parent post was created in another thread"
			httputils.Respond(w, http.StatusConflict, mes)
			_ = tx.Rollback()
			return
		}
	}

	for i, item := range posts {
		row, _ := tx.Query("selectUser", item.Author)
		if !row.Next() {
			mes := models.Message{}
			mes.Message = "Can't find post author by nickname: " + item.Author
			httputils.Respond(w, http.StatusNotFound, mes)
			_ = tx.Rollback()
			return
		}

		row.Close()

		item.Thread = info.Id
		item.Forum = info.Forum

		values += fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
			i*6+1, i*6+2, i*6+3, i*6+4, i*6+5, i*6+6)
		args = append(args, item.Parent, item.Author, item.Message, item.Forum, item.Thread, create)
		if i != l {
			values += ","
		}
	}

	query := "INSERT INTO forum.post(parent, author, message, forum, thread, created) VALUES " + values + " RETURNING id, parent, author, message, isEdited, forum, thread, created"
	posts = []models.Post{}
	row, err := tx.Query(query, args...)

	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	for row.Next() {
		p := models.Post{}
		err = row.Scan(
			&p.Id,
			&p.Parent,
			&p.Author,
			&p.Message,
			&p.IsEdited,
			&p.Forum,
			&p.Thread,
			&p.Created)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		posts = append(posts, p)
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
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
		err = h.conn.QueryRow( "selectThreadBySlug", thread).Scan(
			&result.Id, &result.Title, &result.Author, &result.Forum, &result.Message, &result.Votes, &result.Slug, &result.Created)
	} else {
		err = h.conn.QueryRow( "selectThreadById", isId).Scan(
			&result.Id, &result.Title, &result.Author,  &result.Forum, &result.Message, &result.Votes, &result.Slug, &result.Created)
	}

	if err != nil {
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
	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	if isId == -1 {
		err = tx.QueryRow("updateThreadBySlug",
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
		err = tx.QueryRow("updateThreadById",
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	row, _ := tx.Query("checkUser", vote.Nickname)
	if !row.Next() {
		mes := models.Message{}
		mes.Message = "Can't find user by nickname: " + vote.Nickname
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusNotFound, mes)
		return
	}

	row.Close()


	var result models.Thread
	if isId == -1 {
		err = tx.QueryRow( "selectThreadBySlug", thread).Scan(
			&result.Id, &result.Title, &result.Author, &result.Forum, &result.Message, &result.Votes, &result.Slug, &result.Created)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by slug: " + thread
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	} else {
		err = tx.QueryRow( "selectThreadById", isId).Scan(
			&result.Id, &result.Title, &result.Author, &result.Forum, &result.Message, &result.Votes, &result.Slug, &result.Created)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by id: " + thread
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	}

	vote.Thread = result.Id

	var vot int
	_, err = tx.Exec("insertVote", vote.Thread, vote.Nickname, vote.Voice)
	if err != nil {
		_ = tx.Rollback()
		tx, err = h.conn.Begin()
		if err != nil {
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		err = tx.QueryRow( "selectVote", vote.Thread, vote.Nickname).Scan(&vot)
		if vot != vote.Voice {
			_, err = tx.Exec("updateVote", vote.Thread, vote.Nickname, vote.Voice)
		}
	}

	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	result.Votes = result.Votes - vot + vote.Voice

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
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

	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	var id int
	if isId != -1 {
		id = isId
		err := tx.QueryRow("selectIdThreadById", isId).Scan(&id)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by id: " + thread
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	} else {
		err = tx.QueryRow( "selectIdThreadBySlug", thread).Scan(&id)
		if err != nil {
			mes := models.Message{}
			mes.Message = "Can't find thread by slug: " + thread
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusNotFound, mes)
			return
		}
	}

	var posts []models.Post
	var row *pgx.Rows

	// Float
	switch sort {
	case "tree":
		if since == 0 {
			if desc {
				row, err = tx.Query(
					"treeDesc",
							id, limit)
			} else {
				row, err = tx.Query(
					"tree",
					id, limit)
			}
		} else {
			if desc {
				row, err = tx.Query(
					"treeDescSince",
					id, limit, since)
			} else {
				row, err = tx.Query(
					"treeSince",
					id, limit, since)
			}
		}
	case "parent_tree":
		if since == 0 {
			if desc {
				row, err = tx.Query(
					"parentTreeDesc",
							id, limit)
			} else {
				row, err = tx.Query(
					"parentTree",
							id, limit)
			}
		} else {
			if desc {
				row, err = tx.Query(
					"parentTreeDescSince",
							id, limit, since)
			} else {
				row, err = tx.Query(
					"parentTreeSince",
					id, limit, since)
			}
		}
	default:
		if since == 0 {
			if desc {
				row, err = tx.Query(
					"flatDesc",
					id,
					limit,
				)
			} else {
				row, err = tx.Query(
					"flat",
					id,
					limit,
				)
			}
		} else {
			if desc {
				row, err = tx.Query(
					"flatDescSince",
					id,
					limit,
					since,
				)
			} else {
				row, err = tx.Query(
					"flatSince",
					id,
					limit,
					since,
				)
			}
		}
	}

	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	for row.Next() {
		p := models.Post{}
		err = row.Scan(
			&p.Id,
			&p.Author,
			&p.Created,
			&p.Forum,
			&p.IsEdited,
			&p.Message,
			&p.Parent,
			&p.Thread)
		if err != nil {
			_ = tx.Rollback()
			httputils.Respond(w, http.StatusInternalServerError, nil)
			return
		}

		posts = append(posts, p)
	}

	if posts == nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusOK, []models.Post{})
		return
	}

	err = tx.Commit()
	if err != nil {
		_ = tx.Rollback()
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	httputils.Respond(w, http.StatusOK, posts)
}

// SERVICE

func (h *Handlers) AllClear(w http.ResponseWriter, r *http.Request) {
	tx, err := h.conn.Begin()
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		return
	}

	_, err = tx.Exec("delForum")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec("delPost")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec("delThread")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}
	_, err = tx.Exec("delUser")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	_, err = tx.Exec("delVote")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

	_, err = tx.Exec("delForumUsers")
	if err != nil {
		httputils.Respond(w, http.StatusInternalServerError, nil)
		_ = tx.Rollback()
		return
	}

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

	err := h.conn.QueryRow("countUser").Scan(&status.User)
	if err != nil {
		status.User = 0
	}
	err = h.conn.QueryRow("countForum").Scan(&status.Forum)
	if err != nil {
		status.Forum = 0
	}
	err = h.conn.QueryRow("countThread").Scan(&status.Thread)
	if err != nil {
		status.Thread = 0
	}
	err = h.conn.QueryRow("countPost").Scan(&status.Post)
	if err != nil {
		status.Post = 0
	}

	httputils.Respond(w, http.StatusOK, status)
}

// PREPARE

func (h *Handlers) Prepare() {
	_, _ = h.conn.Prepare("insertVote", "INSERT INTO forum.vote(thread, nickname, voice) VALUES ($1, $2, $3)")
	_, _ = h.conn.Prepare("updateVote", "UPDATE forum.vote SET voice = $3 WHERE thread = $1 and nickname = $2")
	_, _ = h.conn.Prepare("selectVote", "SELECT voice FROM forum.vote WHERE thread = $1 and nickname = $2 LIMIT 1")


	_, _ = h.conn.Prepare("insertUser", "INSERT INTO forum.\"user\"(nickname, fullname, about, email) VALUES ($1, $2, $3, $4)")
	_, _ = h.conn.Prepare("selectDublicateUser", "SELECT nickname, fullname, about, email FROM forum.\"user\" WHERE nickname = $1 OR email = $2 LIMIT 2")
	_, _ = h.conn.Prepare("selectUser", "SELECT nickname, fullname, about, email FROM forum.\"user\" WHERE nickname = $1 LIMIT 1")
	_, _ = h.conn.Prepare("checkUser", "SELECT nickname FROM forum.\"user\" WHERE nickname = $1 LIMIT 1")
	_, _ = h.conn.Prepare("changeUser", "UPDATE forum.\"user\" \n\t\t\t   SET fullname = COALESCE(NULLIF($1, ''), fullname),\n\t\t\t       about = COALESCE(NULLIF($2, ''), about),\n\t\t\t       email = COALESCE(NULLIF($3, ''), email) \n\t\t\t   WHERE nickname = $4 \n\t\t\t   RETURNING nickname, fullname, about, email")
	_, _ = h.conn.Prepare("selectUserOrderDesc", "select nickname, fullname, about, email\n\t\t\t\t\t\tfrom forum.forum_users\n\t\t\t\t\t\tWHERE forum = $1\n\t\t\t\t\t\torder by nickname desc\n\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectUserOrder", "select nickname, fullname, about, email\n\t\t\t\t\t\tfrom forum.forum_users\n\t\t\t\t\t\tWHERE forum = $1\n\t\t\t\t\t\torder by nickname\n\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectUserWhereOrderDesc", "select nickname, fullname, about, email\n\t\t\t\t\t\tfrom forum.forum_users\n\t\t\t\t\t\tWHERE forum = $1 and nickname < $3\n\t\t\t\t\t\torder by nickname desc\n\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectUserWhereOrder", "select nickname, fullname, about, email\n\t\t\t\t\t\tfrom forum.forum_users\n\t\t\t\t\t\tWHERE forum = $1 and nickname > $3\n\t\t\t\t\t\torder by nickname\n\t\t\t\t\tlimit $2")


	_, _ = h.conn.Prepare("insertForum", "INSERT INTO forum.forum(title, \"user\", slug)\n\t\t\t   VALUES ($1, $2, $3)")
	_, _ = h.conn.Prepare("selectForum", "SELECT title, \"user\", slug, posts, threads FROM forum.forum WHERE slug = $1 LIMIT 1")
	_, _ = h.conn.Prepare("checkForum", "SELECT slug FROM forum.forum WHERE slug = $1 LIMIT 1")


	_, _ = h.conn.Prepare("insertThread", "INSERT INTO forum.thread(title, author, forum, message, votes, slug, created)\n\t\tVALUES ($1, $2, $3, $4, $5, nullif($6, ''), $7)\n\t\tRETURNING id")
	_, _ = h.conn.Prepare("selectThread", "SELECT id, title, author, forum, message, votes, slug, created\n\t\t\t\t\tFROM forum.thread\n\t\t\t\t\tWHERE slug = $1 LIMIT 1")
	_, _ = h.conn.Prepare("selectThreadById", "SELECT id, title, author, forum, message, votes, coalesce(slug, '') as slug, created FROM forum.thread WHERE id = $1 LIMIT 1")
	_, _ = h.conn.Prepare("selectThreadOrderDesc", "select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created\n\t\t\t\t\t\tfrom forum.thread t\n\t\t\t\t\t\twhere t.forum = $1\n\t\t\t\t\t\torder by t.created desc\n\t\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectThreadOrder", "select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created\n\t\t\t\t\t\tfrom forum.thread t\n\t\t\t\t\t\twhere t.forum = $1\n\t\t\t\t\t\torder by t.created\n\t\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectThreadWhereOrderDesc", "select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created\n\t\t\t\t\t\tfrom forum.thread t\n\t\t\t\t\t\twhere t.forum = $1 and t.created <= $3\n\t\t\t\t\t\torder by t.created desc\n\t\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectThreadWhereOrder", "select t.id, t.title, t.author, t.forum, t.message, t.votes, coalesce(t.slug, '') as slug, t.created\n\t\t\t\t\t\tfrom forum.thread t\n\t\t\t\t\t\twhere t.forum = $1 and t.created >= $3\n\t\t\t\t\t\torder by t.created\n\t\t\t\t\t\tlimit $2")
	_, _ = h.conn.Prepare("selectIdForumThreadBySlug", "SELECT id, forum FROM forum.thread WHERE slug = $1 LIMIT 1")
	_, _ = h.conn.Prepare("selectIdForumThreadById", "SELECT id, forum FROM forum.thread WHERE id = $1 LIMIT 1")
	_, _ = h.conn.Prepare("selectThreadBySlug", "SELECT id, title, author, forum, message, votes, coalesce(slug, ''), created FROM forum.thread WHERE slug = $1 LIMIT 1")
	_, _ = h.conn.Prepare("updateThreadBySlug", "UPDATE forum.thread SET title = COALESCE(nullif($1, ''), title), message = COALESCE(nullif($2, ''), message) WHERE slug = $3 RETURNING *")
	_, _ = h.conn.Prepare("updateThreadById", "UPDATE forum.thread SET title = COALESCE(nullif($1, ''), title), message = COALESCE(nullif($2, ''), message) WHERE id = $3 RETURNING *")
	_, _ = h.conn.Prepare("selectIdThreadById", "SELECT id as thread FROM forum.thread WHERE id = $1 LIMIT 1")
	_, _ = h.conn.Prepare("selectIdThreadBySlug", "SELECT id as thread FROM forum.thread WHERE slug = $1 LIMIT 1")


	_, _ = h.conn.Prepare("selectPost", "SELECT id, parent, author, message, isEdited, forum, thread, created FROM forum.post WHERE id = $1 LIMIT 1")
	_, _ = h.conn.Prepare("updatePost", "UPDATE forum.post\n\t\t\t\tSET message = COALESCE(nullif($1, ''), message), isEdited = CASE $1 WHEN message THEN false WHEN '' THEN false ELSE true end\n\t\t\t\tWHERE id = $2\n\t\t\t\tRETURNING id, parent, author, message, isEdited, forum, thread, created ")
	_, _ = h.conn.Prepare("selectThreadIdFromPost", "SELECT thread FROM forum.post WHERE id = $1")
	_, _ = h.conn.Prepare("treeDesc", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE thread = $1\n\t\t\t\t\t\t\tORDER BY path DESC, id DESC\n\t\t\t\t\t\t\tLIMIT $2")
	_, _ = h.conn.Prepare("tree", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE thread = $1\n\t\t\t\t\t\t\tORDER BY path, id\n\t\t\t\t\t\t\tLIMIT $2")
	_, _ = h.conn.Prepare("treeDescSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE thread = $1 and path < (SELECT path FROM forum.post WHERE id = $3 LIMIT 1)\n\t\t\t\t\t\t\tORDER BY path DESC, id DESC\n\t\t\t\t\t\t\tLIMIT $2")
	_, _ = h.conn.Prepare("treeSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE thread = $1 and path > (SELECT path FROM forum.post WHERE id = $3 LIMIT 1)\n\t\t\t\t\t\t\tORDER BY path, id\n\t\t\t\t\t\t\tLIMIT $2")
	_, _ = h.conn.Prepare("parentTreeDesc", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE path[1] IN (\n\t\t\t\t\t\t\t\tSELECT id\n\t\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\t\tWHERE thread = $1 and parent = 0\n\t\t\t\t\t\t\t\tORDER BY id DESC\n\t\t\t\t\t\t\t\tLIMIT $2)\n\t\t\t\t\t\t\tORDER BY path[1] DESC, path, id")
	_, _ = h.conn.Prepare("parentTree", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE path[1] IN (\n\t\t\t\t\t\t\t\tSELECT id\n\t\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\t\tWHERE thread = $1 AND parent = 0\n\t\t\t\t\t\t\t\tORDER BY id\n\t\t\t\t\t\t\t\tLIMIT $2)\n\t\t\t\t\t\t\tORDER BY path")
	_, _ = h.conn.Prepare("parentTreeDescSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE path[1] IN (\n\t\t\t\t\t\t\t\tSELECT id\n\t\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\t\tWHERE thread = $1 AND parent = 0 and path[1] < (SELECT path[1] FROM forum.post WHERE id = $3 LIMIT 1)\n\t\t\t\t\t\t\t\tORDER BY id DESC\n\t\t\t\t\t\t\t\tLIMIT $2)\n\t\t\t\t\t\t\tORDER BY path[1] DESC, path, id")
	_, _ = h.conn.Prepare("parentTreeSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\tWHERE path[1] in (\n\t\t\t\t\t\t\t\tSELECT id\n\t\t\t\t\t\t\t\tFROM forum.post\n\t\t\t\t\t\t\t\tWHERE thread = $1 AND parent = 0 and path[1] > (SELECT path[1] FROM forum.post WHERE id = $3 LIMIT 1)\n\t\t\t\t\t\t\t\tORDER BY id ASC\n\t\t\t\t\t\t\t\tLIMIT $2)\n\t\t\t\t\t\t\tORDER BY path, id")
	_, _ = h.conn.Prepare("flatDesc", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t   FROM forum.post\n\t\t\t\t\t   WHERE thread = $1\n\t\t\t\t\t   ORDER BY id DESC\n\t\t\t\t\t   LIMIT $2")
	_, _ = h.conn.Prepare("flat", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t   FROM forum.post\n\t\t\t\t\t   WHERE thread = $1\n\t\t\t\t\t   ORDER BY id\n\t\t\t\t\t   LIMIT $2")
	_, _ = h.conn.Prepare("flatDescSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t   FROM forum.post\n\t\t\t\t\t   WHERE thread = $1 and id < $3\n\t\t\t\t\t   ORDER BY id DESC\n\t\t\t\t\t   LIMIT $2")
	_, _ = h.conn.Prepare("flatSince", "SELECT id, author, created, forum, isEdited, message, parent, thread\n\t\t\t\t\t   FROM forum.post\n\t\t\t\t\t   WHERE thread = $1 and id > $3\n\t\t\t\t\t   ORDER BY id\n\t\t\t\t\t   LIMIT $2")


	_, _ = h.conn.Prepare("delForum", "TRUNCATE forum.forum CASCADE")
	_, _ = h.conn.Prepare("delPost", "TRUNCATE forum.post CASCADE")
	_, _ = h.conn.Prepare("delThread", "TRUNCATE forum.thread CASCADE")
	_, _ = h.conn.Prepare("delUser", "TRUNCATE forum.\"user\" CASCADE")
	_, _ = h.conn.Prepare("delVote", "TRUNCATE forum.vote CASCADE")
	_, _ = h.conn.Prepare("delForumUsers", "TRUNCATE forum.forum_users CASCADE")
	_, _ = h.conn.Prepare("countUser", "SELECT COUNT(*) FROM forum.\"user\"")
	_, _ = h.conn.Prepare("countForum", "SELECT COUNT(*) FROM forum.forum")
	_, _ = h.conn.Prepare("countThread", "SELECT COUNT(*) FROM forum.thread")
	_, _ = h.conn.Prepare("countPost", "SELECT COUNT(*) FROM forum.post")
}
