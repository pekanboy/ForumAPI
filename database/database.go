package database

import (
	_ "github.com/jackc/pgx/stdlib"
	"github.com/jmoiron/sqlx"
)

type Postgres struct {
	db *sqlx.DB
}

func NewPostgres(dsn string) (*Postgres, error) {
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		db: db,
	}, nil
}

func (p *Postgres) GetPostgres() *sqlx.DB {
	return p.db
}

func (p *Postgres) Close() error {
	err := p.db.Close()
	return err
}
