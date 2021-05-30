package models

type Status struct {
	User   int `json:"user" db:"user"`
	Forum  int `json:"forum" db:"forum"`
	Thread int `json:"thread" db:"thread"`
	Post   int `json:"post" db:"post"`
}
