package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"server/database"
	handlers "server/handlers"
)

func main() {
	postgres, err := database.NewPostgres("user=postgres dbname=postgres password=admin host=127.0.0.1 port=5432 sslmode=disable")

	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter()

	handler := handlers.NewHandler(postgres.GetPostgres())

	user := router.PathPrefix("/user").Subrouter()
	user.HandleFunc("/{nickname}/create", handler.CreateUser).Methods(http.MethodPost)
	user.HandleFunc("/{nickname}/profile", handler.GetUser).Methods(http.MethodGet)
	user.HandleFunc("/{nickname}/profile", handler.ChangeUser).Methods(http.MethodPost)

	forum := router.PathPrefix("/forum").Subrouter()
	forum.HandleFunc("/create", handler.CreateForum).Methods(http.MethodPost)
	forum.HandleFunc("/{slug}/details", handler.GetForum).Methods(http.MethodGet)
	forum.HandleFunc("/{slug}/create", handler.CreateThread).Methods(http.MethodPost)
	forum.HandleFunc("/{slug}/users", handler.GetForumUsers).Methods(http.MethodGet)
	forum.HandleFunc("/{slug}/threads", handler.GetForumThreads).Methods(http.MethodGet)

	post := router.PathPrefix("/post").Subrouter()
	post.HandleFunc("/{id}/details", handler.GetPost).Methods(http.MethodGet)
	post.HandleFunc("/{id}/details", handler.ChangePost).Methods(http.MethodPost)

	thread := router.PathPrefix("/thread").Subrouter()
	thread.HandleFunc("/{slug_or_id}/create", handler.CreatePost).Methods(http.MethodPost)
	thread.HandleFunc("/{slug_or_id}/details", handler.GetThread).Methods(http.MethodGet)
	thread.HandleFunc("/{slug_or_id}/details", handler.ChangeThread).Methods(http.MethodPost)
	thread.HandleFunc("/{slug_or_id}/vote", handler.CreateVote).Methods(http.MethodPost)
	thread.HandleFunc("/{slug_or_id}/posts", handler.ThreadPosts).Methods(http.MethodGet)

	service := router.PathPrefix("/service").Subrouter()
	service.HandleFunc("/clear", handler.AllClear).Methods(http.MethodPost)
	service.HandleFunc("/status", handler.AllInfo).Methods(http.MethodGet)

	server := &http.Server{
		Handler: router,
		Addr:    ":5000",
	}

	log.Println("Server starting")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
