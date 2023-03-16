package database

import (
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"

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

func (db *DB) InsertFile(file *File) (File, error) {
	if _, err := db.GetFileByHash(file.Adler32, file.Sha256); err != sql.ErrNoRows {
		// TODO: 如何防止重复文件？
		// 同仓库同文件名不行, hash相同也不行
	}
	if err := insertFile(db.DB, file); err != nil {
		return File{}, err
	}
	return db.GetFileByHash(file.Adler32, file.Sha256)
}

// CheckSameFile 发现有相同文件 (同名或同内容) 时返回错误,
// 未发现相同文件则返回 nil 或其他错误.
func (db *DB) CheckSameFile(file *File) error {
	if err := db.checkSameFileName(file); err != nil {
		return err
	}

	return nil // TODO
}

// 有同名文件时返回 error(同名文件已存在), 无同名文件则返回 nil 或其他错误.
func (db *DB) checkSameFileName(file *File) error {
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
	files, err := db.GetFilesByAlder32(file.Adler32)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if len(files) == 0 {
		return err
	}
	idList := model.FilesToString(files)
	if len(files) == 1 {
		sameFile := files[0]
		sameFilePath := filepath.Join(db.ProjectPath, sameFile.BucketID, sameFile.Name)
	}
}

func (db *DB) GetFilesByAlder32(adler32 string) ([]File, error) {
	rows, err := db.Query(stmt.GetFileByHash, adler32)
	if err != nil {
		return nil, err
	}
	return scanFiles(rows)
}

func (db *DB) GetFileByHash(adler32, sha256 string) (File, error) {
	row := db.QueryRow(stmt.GetFileByHash, adler32, sha256)
	return scanFile(row)
}

func (db *DB) GetFileByName(name string) (File, error) {
	row := db.QueryRow(stmt.GetFileByName, name)
	return scanFile(row)
}
