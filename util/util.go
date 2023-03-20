package util

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

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

// MustMkdir 创建資料夹, 如果 perm 等于零, 则使用默认权限.
func MustMkdir(name string, perm fs.FileMode) {
	if perm == 0 {
		perm = NormalFolerPerm
	}
	lo.Must0(os.Mkdir(name, perm))
}

// MkdirIfNotExists 创建資料夹, 忽略 ErrExist, 如果 perm 等于零, 则使用默认权限.
func MkdirIfNotExists(name string, perm fs.FileMode) {
	if perm == 0 {
		perm = NormalFolerPerm
	}
	if PathIsExist(name) {
		return
	}
	lo.Must0(os.Mkdir(name, perm))
}

// WriteFile 写檔案, 如果 perm 等于零, 则使用默认权限.
func WriteFile(name string, data []byte, perm fs.FileMode) error {
	if perm == 0 {
		perm = NormalFilePerm
	}
	return os.WriteFile(name, data, perm)
}

// WriteReadonlyFile 写檔案, 同时把檔案设为只读.
func WriteReadonlyFile(name string, data []byte) error {
	return os.WriteFile(name, data, ReadonlyFilePerm)
}

func WriteTOML(data interface{}, filename string) {
	dataTOML := lo.Must(toml.Marshal(data))
	lo.Must0(WriteFile(filename, dataTOML, 0))
}

// WriteJSON 把 data 转换为漂亮格式的 JSON 并写入檔案 filename 中。
func WriteJSON(data interface{}, filename string) {
	dataJSON := lo.Must(json.MarshalIndent(data, "", "    "))
	lo.Must0(WriteFile(filename, dataJSON, 0))
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

func IsRegularFile(name string) (ok bool, err error) {
	info, err := os.Stat(name)
	if err != nil {
		return
	}
	return info.Mode().IsRegular(), nil
}

func GetRegularFiles(folder string) (files []string, err error) {
	pattern := filepath.Join(folder, "*")
	items, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	for _, file := range items {
		ok, err := IsRegularFile(file)
		if err != nil {
			return nil, err
		}
		if ok {
			files = append(files, file)
		}
	}
	return files, nil
}

// UnlockFolder 把資料夹设为可访问, 可添加/删除檔案.
func UnlockFolder(name string) {
	lo.Must0(os.Chmod(name, NormalFolerPerm))
}

// LockFolder 把資料夹设为只读权限 (不可添加/删除檔案)
func LockFolder(name string) {
	lo.Must0(os.Chmod(name, ReadonlyFolderPerm))
}

// UnlockFile 把檔案设为可读写.
func UnlockFile(name string) {
	lo.Must0(os.Chmod(name, NormalFilePerm))
}

// LockFile 把檔案设为只读权限 (不可写)
func LockFile(name string) {
	lo.Must0(os.Chmod(name, ReadonlyFilePerm))
}

// BLAKE2b is faster than MD5, SHA-1, SHA-2, and SHA-3, on 64-bit x86-64 and ARM architectures.
// https://en.wikipedia.org/wiki/BLAKE_(hash_function)#BLAKE2
// https://blog.min.io/fast-hashing-in-golang-using-blake2/
// https://pkg.go.dev/crypto/sha256#example-New-File
func FileSum512(name string) (HexString, error) {
	f, err := os.Open(name)
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

// https://stackoverflow.com/questions/30376921/how-do-you-copy-a-file-in-go
func CopyFile(destPath, sourcePath string) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err1 := io.Copy(dst, src)
	err2 := dst.Sync()
	return WrapErrors(err1, err2)
}

func DeleteFiles(files []string) (err error) {
	for _, file := range files {
		e := os.Remove(file)
		err = WrapErrors(err, e)
	}
	return err
}
