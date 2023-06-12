package database

import (
	"database/sql"
	"time"

	"github.com/ahui2016/local-buckets/model"
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

// getInt1 gets one Integer value from the database.
func getInt1(tx TX, query string, arg ...any) (n int64, err error) {
	row := tx.QueryRow(query, arg...)
	err = row.Scan(&n)
	return
}

// txGetFileByID 可能可以删除
func txGetFileByID(tx TX, id int64) (File, error) {
	row := tx.QueryRow(stmt.GetFileByID, id)
	return ScanFile(row)
}

func insertBucket(tx TX, b *Bucket) error {
	_, err := tx.Exec(
		stmt.InsertBucket,
		// b.ID, 自增ID
		b.Name,
		b.Title,
		b.Subtitle,
		b.Encrypted,
	)
	return err
}

func insertBucketWithID(tx TX, b *Bucket) error {
	_, err := tx.Exec(
		stmt.InsertBucketWithID,
		b.ID,
		b.Name,
		b.Title,
		b.Subtitle,
		b.Encrypted,
	)
	return err
}

func scanBucket(row Row) (b Bucket, err error) {
	err = row.Scan(
		&b.ID,
		&b.Name,
		&b.Title,
		&b.Subtitle,
		&b.Encrypted,
	)
	return
}

func scanBuckets(rows *sql.Rows) (all []*Bucket, err error) {
	for rows.Next() {
		b, err := scanBucket(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, &b)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}

func scanKeywords(rows *sql.Rows) (all []string, err error) {
	var kw string
	for rows.Next() {
		if err := rows.Scan(&kw); err != nil {
			return nil, err
		}
		all = append(all, kw)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}

func insertFile(tx TX, f *File) error {
	_, err := tx.Exec(
		stmt.InsertFile,
		// f.ID, 自增ID
		f.Checksum,
		f.BucketName,
		f.Name,
		f.Notes,
		f.Keywords,
		f.Size,
		f.Type,
		f.Like,
		f.CTime,
		f.UTime,
		f.Checked,
		f.Damaged,
		f.Deleted,
	)
	return err
}

// insertFileWithID 主要用于复制文档到备份仓库.
func insertFileWithID(tx TX, f *File) error {
	_, err := tx.Exec(
		stmt.InsertFileWithID,
		f.ID,
		f.Checksum,
		f.BucketName,
		f.Name,
		f.Notes,
		f.Keywords,
		f.Size,
		f.Type,
		f.Like,
		f.CTime,
		f.UTime,
		f.Checked,
		f.Damaged,
		f.Deleted,
	)
	return err
}

func ScanFile(row Row) (f File, err error) {
	err = row.Scan(
		&f.ID,
		&f.Checksum,
		&f.BucketName,
		&f.Name,
		&f.Notes,
		&f.Keywords,
		&f.Size,
		&f.Type,
		&f.Like,
		&f.CTime,
		&f.UTime,
		&f.Checked,
		&f.Damaged,
		&f.Deleted,
	)
	return
}

func scanFilePlus(row Row) (f FilePlus, err error) {
	err = row.Scan(
		&f.ID,
		&f.Checksum,
		&f.BucketName,
		&f.Name,
		&f.Notes,
		&f.Keywords,
		&f.Size,
		&f.Type,
		&f.Like,
		&f.CTime,
		&f.UTime,
		&f.Checked,
		&f.Damaged,
		&f.Deleted,
		&f.Encrypted,
	)
	return
}

func scanFiles(rows *sql.Rows) (all []*File, err error) {
	for rows.Next() {
		f, err := ScanFile(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, &f)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}

func scanFilesPlus(rows *sql.Rows) (all []*FilePlus, err error) {
	for rows.Next() {
		f, err := scanFilePlus(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, &f)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}

func getFiles(tx TX, query string, args ...any) (files []*File, err error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return
	}
	return scanFiles(rows)
}

func getFilesPlus(tx TX, query string, args ...any) (files []*FilePlus, err error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return
	}
	return scanFilesPlus(rows)
}

func countFilesNeedCheck(tx TX, interval int64) (int64, error) {
	now := time.Now().Unix()
	interval = interval * 24 * 60 * 60 // 单位 "日" 转为 "秒"
	needCheckDate := time.Unix(now-interval, 0).Format(model.RFC3339)
	return getInt1(tx, stmt.CountFilesNeedCheck, needCheckDate)
}
