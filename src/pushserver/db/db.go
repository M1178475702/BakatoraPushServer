package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type DB struct {
	db     *sqlx.DB
	Client ClientDb
}

func New(user, pwd, host, port, database string) *DB{
	db := new(DB)
	url := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4",user, pwd, host, port, database)
	db.db = sqlx.MustConnect("mysql", url)
	db.Client = &ClientDbImpl{db: db.db}
	return db
}
