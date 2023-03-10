package database

import (
	"database/sql"

	"github.com/ahui2016/local-buckets/stmt"
	"github.com/ahui2016/local-buckets/util"
)

type TX interface {
	Exec(string, ...any) (sql.Result, error)
	Query(string, ...any) (*sql.Rows, error)
	QueryRow(string, ...any) *sql.Row
}

type Row interface {
	Scan(...any) error
}

func insertBucket(tx TX, b *Bucket) error {
	_, err := tx.Exec(
		stmt.InsertBucket,
		b.ID,
		b.Title,
		b.Subtitle,
		b.Capacity,
		b.MaxFilesize,
		b.Encrypted,
	)
	return err
}

func scanBucket(row Row) (b Bucket, err error) {
	err = row.Scan(
		&b.ID,
		&b.Title,
		&b.Subtitle,
		&b.Capacity,
		&b.MaxFilesize,
		&b.Encrypted,
	)
	return
}

func scanBuckets(rows *sql.Rows) (all []Bucket, err error) {
	for rows.Next() {
		b, err := scanBucket(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, b)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}
