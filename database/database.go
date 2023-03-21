package database

import (
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/ahui2016/local-buckets/util"
	"github.com/samber/lo"
	_ "modernc.org/sqlite"
)

type (
	Base64String     = string
	Bucket           = model.Bucket
	File             = model.File
	ErrSameNameFiles = model.ErrSameNameFiles
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

func (db *DB) mustBegin() *sql.Tx {
	return lo.Must(db.DB.Begin())
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

// GetInt1 gets one Integer value from the database.
func (db *DB) GetInt1(query string, arg ...any) (int64, error) {
	return getInt1(db.DB, query, arg...)
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

func (db *DB) GetBucket(id string) (Bucket, error) {
	row := db.QueryRow(stmt.GetBucket, id)
	return scanBucket(row)
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

// InsertFile 主要用于同名檔案冲突时的逐一处理.
func (db *DB) InsertFile(file *File) (*File, error) {
	if err := insertFile(db.DB, file); err != nil {
		return nil, err
	}
	f, err := db.GetFileByChecksum(file.Checksum)
	return &f, err
}

// InsertFiles inserts files into the database.
// 注意, 在使用该函数之前, 请先使用 db.CheckSameFiles() 检查全部等待处理的檔案.
func (db *DB) InsertFiles(files []*File) error {
	tx := db.mustBegin()
	defer tx.Rollback()

	for _, file := range files {
		if err := insertFile(db.DB, file); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// CheckSameFiles 检查有无同名/相同内容的檔案,
// 发现相同内容的檔案时, 记录全部重复檔案,
// 但发现同名檔案时, 则立即返回错误 (因为前端需要对同名檔案进行逐个处理).
func (db *DB) CheckSameFiles(files []*File) (allErr error) {
	for _, file := range files {
		if err := db.checkSameFile(file); err != nil {
			if e, ok := err.(ErrSameNameFiles); ok {
				return e
			}
			allErr = util.WrapErrors(allErr, err)
		}
	}
	return
}

// CheckSameFile 发现有相同檔案 (同名或同内容) 时返回错误,
// 未发现相同檔案则返回 nil 或其他错误.
func (db *DB) checkSameFile(file *File) error {
	if err := db.CheckSameFilename(file.Name); err != nil {
		return err
	}
	return db.checkSameChecksum(file)
}

// 有同名檔案时返回 ErrSameNameFiles, 无同名檔案则返回 nil 或其他错误.
func (db *DB) CheckSameFilename(name string) error {
	same, err := db.GetFileByName(name)
	if err == nil && len(same.Name) > 0 {
		return model.NewErrSameNameFiles(same)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return err
}

// 有相同内容的檔案时返回 error(相同内容的檔案已存在),
// 无相同内容的檔案则返回 nil 或其他错误.
func (db *DB) checkSameChecksum(file *File) error {
	same, err := db.GetFileByChecksum(file.Checksum)
	if err == nil && len(same.Name) > 0 {
		return fmt.Errorf("相同内容的檔案已存在: %s ↔ %s", file.Name, same.Name)
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
