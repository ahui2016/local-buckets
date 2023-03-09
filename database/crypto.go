package database

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/samber/lo"
)

const (
	KeySize         = 16
	NonceSize       = 12
	BcryptCost      = 15 // 根据服务器运算速度而定, 由于我这个是本地程序, 因此设大一点
	DefaultPassword = "abc123"
)

type (
	HexString = string
	Nonce     = [NonceSize]byte
	SecretKey = [KeySize]byte
)

var (
	ErrWrongPassword = errors.New("wrong password (密碼錯誤)")
)

// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
func newNonce() (nonce Nonce, err error) {
	_, err = rand.Read(nonce[:])
	return
}

func randomKey() (key SecretKey, err error) {
	_, err = rand.Read(key[:])
	return
}

func encrypt(data []byte, aesgcm cipher.AEAD) (encrypted []byte, err error) {
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}
	encrypted = aesgcm.Seal(nonce[:], nonce[:], data, nil)
	return
}

func decrypt(ciphertext HexString, aesgcm cipher.AEAD) (data []byte, err error) {
	blob := lo.Must(hex.DecodeString(ciphertext))
	nonce := blob[:NonceSize]
	encryped := blob[NonceSize:]
	return aesgcm.Open(nil, nonce, encryped, nil)
}

func newGCM(password string) cipher.AEAD {
	key := md5.Sum([]byte(password))
	block := lo.Must(aes.NewCipher(key[:]))
	return lo.Must(cipher.NewGCM(block))
}

// DefaultCipherKey 用默認密碼去加密真正的密鑰.
func DefaultCipherKey() HexString {
	aesgcm := newGCM(DefaultPassword)
	realKey := lo.Must(randomKey())
	encryptedKey := lo.Must(encrypt(realKey[:], aesgcm))
	return hex.EncodeToString(encryptedKey)
}
