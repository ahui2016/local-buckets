package util

import (
	"os"

	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

const (
	PermNormalFile = 0666
)

// GetExePath returns the path name for the executable
// that started the current process.
func GetExePath() string {
	return lo.Must1(os.Executable())
}

func WriteFile(name string, data []byte) error {
	return os.WriteFile(name, data, PermNormalFile)
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
