package database

import (
	"database/sql"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	_ "modernc.org/sqlite"
)

type (
	Bucket = model.Bucket
)

type DB struct {
	Path    string
	DB      *sql.DB
	dbKey   SecretKey
	userKey SecretKey
}

func (db *DB) Exec(query string, args ...any) (err error) {
	_, err = db.DB.Exec(query, args)
	return
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(query, args)
}

func (db *DB) Open(dbPath string) (err error) {
	if db.DB, err = sql.Open("sqlite", dbPath); err != nil {
		return
	}
	db.Path = dbPath
	if err = db.Exec(stmt.CreateTables); err != nil {
		return
	}
	return nil
}

func (db *DB) GetAllBuckets() ([]Bucket, error) {
	rows, err := db.Query(stmt.GetAllBuckets)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}
