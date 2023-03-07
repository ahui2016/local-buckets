package util

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

const (
	NormalFilePerm  = 0666
	NormalFolerPerm = 0750
)

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
