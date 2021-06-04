package database

import (
	"github.com/jackc/pgx"
	_ "github.com/jackc/pgx/stdlib"
)

type Postgres struct {
	conn *pgx.ConnPool
}

func NewPostgres(dsn string) (*Postgres, error) {
	conf := pgx.ConnConfig{
		User:                 "postgres",
		Database:             "postgres",
		Password:             "admin",
		PreferSimpleProtocol: false,
	}

	poolConf := pgx.ConnPoolConfig{
		ConnConfig:     conf,
		MaxConnections: 100,
		AfterConnect:   nil,
		AcquireTimeout: 0,
	}

	conn, err := pgx.NewConnPool(poolConf)
	if err != nil {
		return nil, err
	}
	return &Postgres{
		conn: conn,
	}, nil
}

func (p *Postgres) GetPostgres() *pgx.ConnPool {
	return p.conn
}

func (p *Postgres) Close() error {
	p.conn.Close()
	return nil
}
