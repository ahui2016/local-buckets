package util

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

const (
	NormalFilePerm  = 0666
	NormalFolerPerm = 0750
)

type Base64String = string

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

func MustMkdir(name string) {
	lo.Must0(os.Mkdir(name, NormalFolerPerm))
}

func MkdirIfNotExists(name string) {
	if PathIsExist(name) {
		return
	}
	lo.Must0(os.Mkdir(name, NormalFolerPerm))
}

func WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, NormalFilePerm)
}

func WriteTOML(val interface{}, filename string) {
	data := lo.Must(toml.Marshal(val))
	WriteFile(filename, data)
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
