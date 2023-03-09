package database

import (
	"crypto/cipher"
	"database/sql"
	"encoding/hex"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/samber/lo"
	_ "modernc.org/sqlite"
)

type (
	Base64String = string
	Bucket       = model.Bucket
)

type DB struct {
	Path      string
	DB        *sql.DB
	cipherKey HexString
	aesgcm    cipher.AEAD
}

func (db *DB) Exec(query string, args ...any) (err error) {
	_, err = db.DB.Exec(query, args...)
	return
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}

func (db *DB) Open(dbPath string, cipherKey HexString) (err error) {
	if db.DB, err = sql.Open("sqlite", dbPath); err != nil {
		return
	}
	db.Path = dbPath
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
