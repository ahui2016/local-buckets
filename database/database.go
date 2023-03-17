package database

import (
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/samber/lo"
	_ "modernc.org/sqlite"
)

type (
	Base64String = string
	Bucket       = model.Bucket
	File         = model.File
)

type DB struct {
	Path        string
	ProjectPath string
	DB          *sql.DB
	cipherKey   HexString
	aesgcm      cipher.AEAD
}

func (db *DB) Exec(query string, args ...any) (err error) {
	_, err = db.DB.Exec(query, args...)
	return
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(query, args...)
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}

func (db *DB) Open(dbPath, pjPath string, cipherKey HexString) (err error) {
	const turnOnForeignKeys = "?_pragma=foreign_keys(1)"
	if db.DB, err = sql.Open("sqlite", dbPath+turnOnForeignKeys); err != nil {
		return
	}
	db.Path = dbPath
	db.ProjectPath = pjPath
	if err = db.Exec(stmt.CreateTables); err != nil {
		return
	}
	db.cipherKey = cipherKey
	return nil
}

func (db *DB) SetAESGCM(password string) (realKey []byte, err error) {
	aesgcm := newGCM(password)
	realKey, err = decrypt(db.cipherKey, aesgcm)
	if err == nil {
		db.aesgcm = aesgcm
	}
	return
}

func (db *DB) ChangePassword(oldPwd, newPwd string) (HexString, error) {
	realKey, err := db.SetAESGCM(oldPwd)
	if err != nil {
		return "", err
	}
	aesgcm := newGCM(newPwd)
	encryptedKey := lo.Must(encrypt(realKey[:], aesgcm))
	db.aesgcm = aesgcm
	db.cipherKey = hex.EncodeToString(encryptedKey)
	return db.cipherKey, nil
}

func (db *DB) GetAllBuckets() ([]Bucket, error) {
	rows, err := db.Query(stmt.GetAllBuckets)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}

func (db *DB) InsertBucket(form *model.CreateBucketForm) (bucket *Bucket, err error) {
	bucket, err = model.NewBucket(form)
	if err != nil {
		return
	}
	if err = insertBucket(db.DB, bucket); err != nil {
		return nil, err
	}
	return
}

func (db *DB) InsertFile(file *File) (*File, error) {
	if err := db.CheckSameFile(file); err != nil {
		return nil, err
	}
	if err := insertFile(db.DB, file); err != nil {
		return nil, err
	}
	f, err := db.GetFileByChecksum(file.Checksum)
	return &f, err
}

// CheckSameFile 发现有相同文件 (同名或同内容) 时返回错误,
// 未发现相同文件则返回 nil 或其他错误.
func (db *DB) CheckSameFile(file *File) error {
	if err := db.checkSameFilename(file); err != nil {
		return err
	}
	return db.checkSameChecksum(file)
}

// 有同名文件时返回 error(同名文件已存在), 无同名文件则返回 nil 或其他错误.
func (db *DB) checkSameFilename(file *File) error {
	same, err := db.GetFileByName(file.Name)
	if err == nil && len(same.Name) > 0 {
		return fmt.Errorf("同名文件(檔案)已存在: %s", same.Name)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return err
}

// 有相同内容的文件时返回 error(相同内容的文件已存在),
// 无相同内容的文件则返回 nil 或其他错误.
func (db *DB) checkSameChecksum(file *File) error {
	same, err := db.GetFileByChecksum(file.Checksum)
	if err == nil && len(same.Name) > 0 {
		return fmt.Errorf("相同内容的文件(檔案)已存在: %s", same.Name)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return err
}

func (db *DB) GetFileByChecksum(checksum string) (File, error) {
	row := db.QueryRow(stmt.GetFileByChecksum, checksum)
	return scanFile(row)
}

func (db *DB) GetFileByName(name string) (File, error) {
	row := db.QueryRow(stmt.GetFileByName, name)
	return scanFile(row)
}
