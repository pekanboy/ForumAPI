package models

type Forum struct {
	Title   string `json:"title" db:"title"`
	User    string `json:"user" db:"user"`
	Slug    string `json:"slug" db:"slug"`
	Posts   int    `json:"posts" db:"posts"`
	Threads int    `json:"threads" db:"threads"`
}
