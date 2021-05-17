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

	server := &http.Server{
		Handler: router,
		Addr:    ":5000",
	}

	log.Println("Server starting")
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

}
