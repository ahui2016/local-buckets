package database

import (
	"database/sql"
	"os"
	"path/filepath"
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

func insertBucket(tx TX, b *Bucket) error {
	_, err := tx.Exec(
		stmt.InsertBucket,
		b.ID,
		b.Title,
		b.Subtitle,
		b.Capacity,
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

func insertFile(tx TX, f *File) error {
	_, err := tx.Exec(
		stmt.InsertFile,
		// f.ID, 自增ID
		f.Checksum,
		f.BucketID,
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

func insertFileWithID(tx TX, f *File) error {
	_, err := tx.Exec(
		stmt.InsertFile,
		f.ID,
		f.Checksum,
		f.BucketID,
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

func scanFile(row Row) (f File, err error) {
	err = row.Scan(
		&f.ID,
		&f.Checksum,
		&f.BucketID,
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

func scanFiles(rows *sql.Rows) (all []File, err error) {
	for rows.Next() {
		f, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		all = append(all, f)
	}
	err = util.WrapErrors(rows.Err(), rows.Close())
	return
}

func getFiles(tx TX, query string, args ...any) (files []*File, err error) {
	rows, err := tx.Query(query, args...)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		file, err := scanFile(rows)
		if err != nil {
			return nil, err
		}
		files = append(files, &file)
	}
	err = rows.Err()
	return
}

func countFilesNeedCheck(tx TX, interval int64) (int64, error) {
	now := time.Now().Unix()
	interval = interval * 24 * 60 * 60 // 单位 "日" 转为 "秒"
	needCheckDate := time.Unix(now-interval, 0).Format(model.RFC3339)
	return getInt1(tx, stmt.CountFilesNeedCheck, needCheckDate)
}

// DeleteFile 刪除檔案, 包括從數據庫中刪除和從硬碟中刪除.
func DeleteFile(tx TX, bucketsDir, tempDir string, file *File) error {
	moved := MovedFile{
		Src: filepath.Join(bucketsDir, file.BucketID, file.Name),
		Dst: filepath.Join(tempDir, file.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if _, err := tx.Exec(stmt.DeleteFile, file.ID); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return os.Remove(moved.Dst)
}
