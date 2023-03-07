package database

const (
	KeySize   = 32
	NonceSize = 24
)

type (
	Nonce     = [NonceSize]byte
	SecretKey = [KeySize]byte
)
