package models

import "time"

type Post struct {
	Id       int       `json:"id" db:"id"`
	Parent   int       `json:"parent" db:"parent"`
	Author   string    `json:"author" db:"author"`
	Message  string    `json:"message" db:"message"`
	IsEdited bool      `json:"isEdited" db:"isEdited"`
	Forum    string    `json:"forum" db:"forum"`
	Thread   int       `json:"thread" db:"thread"`
	Created  time.Time `json:"created" db:"created"`
}
