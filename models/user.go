package models

type User struct {
	Nickname string `json:"nickname" db:"nickname"`
	Fullname string `json:"fullname" db:"fullname"`
	About    string `json:"about" db:"about"`
	Email    string `json:"email" db:"email"`
}
