package models

type Vote struct {
	Nickname string `json:"nickname" bd:"nickname"`
	Voice    int    `json:"voice" bd:"voice"`
	Thread   string `bd:"thread"`
}
