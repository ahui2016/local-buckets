package util

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
	"golang.org/x/crypto/blake2b"
)

const (
	ReadonlyFilePerm   = 0555
	ReadonlyFolderPerm = 0550
	NormalFilePerm     = 0666
	NormalFolerPerm    = 0750
)

type (
	Base64String = string
	HexString    = string
)

func Base64Encode(data []byte) Base64String {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(s Base64String) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// WrapErrors 把多个错误合并为一个错误.
func WrapErrors(allErrors ...error) (wrapped error) {
	for _, err := range allErrors {
		if err != nil {
			if wrapped == nil {
				wrapped = err
			} else {
				wrapped = fmt.Errorf("%w | %w", wrapped, err)
			}
		}
	}
	return
}

// GetExePath returns the path name for the executable
// that started the current process.
func GetExePath() string {
	return lo.Must1(os.Executable())
}

// MustMkdir 创建文件夹, 如果 perm 等于零, 则使用默认权限.
func MustMkdir(name string, perm fs.FileMode) {
	if perm == 0 {
		perm = NormalFolerPerm
	}
	lo.Must0(os.Mkdir(name, perm))
}

// MkdirIfNotExists 创建文件夹, 忽略 ErrExist, 如果 perm 等于零, 则使用默认权限.
func MkdirIfNotExists(name string, perm fs.FileMode) {
	if perm == 0 {
		perm = NormalFolerPerm
	}
	if PathIsExist(name) {
		return
	}
	lo.Must0(os.Mkdir(name, perm))
}

// WriteFile 写文件, 如果 perm 等于零, 则使用默认权限.
func WriteFile(name string, data []byte, perm fs.FileMode) error {
	if perm == 0 {
		perm = NormalFilePerm
	}
	return os.WriteFile(name, data, perm)
}

// WriteReadonlyFile 写文件, 同时把文件设为只读.
func WriteReadonlyFile(name string, data []byte) error {
	return os.WriteFile(name, data, ReadonlyFilePerm)
}

func WriteTOML(val interface{}, filename string) {
	data := lo.Must(toml.Marshal(val))
	lo.Must0(WriteFile(filename, data, 0))
}

func PathIsNotExist(name string) (ok bool) {
	_, err := os.Lstat(name)
	if os.IsNotExist(err) {
		ok = true
		err = nil
	}
	lo.Must0(err)
	return
}

func PathIsExist(name string) bool {
	return !PathIsNotExist(name)
}

// UnlockFolder 把文件夹设为可访问, 可添加/删除文件.
func UnlockFolder(name string) {
	lo.Must0(os.Chmod(name, NormalFolerPerm))
}

// LockFolder 把文件夹设为只读权限 (不可添加/删除文件)
func LockFolder(name string) {
	lo.Must0(os.Chmod(name, ReadonlyFolderPerm))
}

// UnlockFile 把文件设为可读写.
func UnlockFile(name string) {
	lo.Must0(os.Chmod(name, NormalFilePerm))
}

// LockFile 把文件设为只读权限 (不可写)
func LockFile(name string) {
	lo.Must0(os.Chmod(name, ReadonlyFilePerm))
}

// BLAKE2b is faster than MD5, SHA-1, SHA-2, and SHA-3, on 64-bit x86-64 and ARM architectures.
// https://en.wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2
// https://blog.min.io/fast-hashing-in-golang-using-blake2/
// https://pkg.go.dev/crypto/sha256#example-New-File
func FileSum512(name string) (HexString, error) {
	f, err := os.Open("file.txt")
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := lo.Must(blake2b.New512(nil))
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	checksum := h.Sum(nil)
	return hex.EncodeToString(checksum), nil
}
