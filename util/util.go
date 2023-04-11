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
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
	"golang.org/x/crypto/blake2b"
)

const (
	ReadonlyFilePerm = 0555
	NormalFilePerm   = 0666
	NormalFolerPerm  = 0750
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

func Mkdir(name string) error {
	return os.Mkdir(name, NormalFolerPerm)
}

// MkdirIfNotExists 创建資料夹, 忽略 ErrExist.
// 在 Windows 里, 文件夹的只读属性不起作用, 为了统一行为, 不把文件夹设为只读.
func MkdirIfNotExists(name string) error {
	if PathExists(name) {
		return nil
	}
	return Mkdir(name)
}

// CreateFile 把 src 的数据写入 filePath, 自动关闭 file.
func CreateFile(filePath string, src io.Reader) error {
	file, err := CreateReturnFile(filePath, src)
	if err == nil {
		file.Close()
	}
	return err
}

// CreateReturnFile 把 src 的数据写入 filePath
// 会自动创建或覆盖文件，返回 file, 要记得关闭资源。
func CreateReturnFile(filePath string, src io.Reader) (*os.File, error) {
	f, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, NormalFilePerm)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(f, src)
	return f, err
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

func WriteTOML(data interface{}, filename string) error {
	dataTOML, err := toml.Marshal(data)
	if err != nil {
		return err
	}
	return WriteFile(filename, dataTOML, 0)
}

// WriteJSON 把 data 转换为漂亮格式的 JSON 并写入檔案 filename 中。
func WriteJSON(data interface{}, filename string) error {
	dataJSON, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		return err
	}
	return WriteFile(filename, dataJSON, 0)
}

// TODO: 改名 PathNotExists
func PathNotExists(name string) (ok bool) {
	_, err := os.Lstat(name)
	if os.IsNotExist(err) {
		ok = true
		err = nil
	}
	lo.Must0(err)
	return
}

func PathExists(name string) bool {
	return !PathNotExists(name)
}

func DirIsEmpty(dirpath string) (ok bool, err error) {
	items, err := filepath.Glob(dirpath + "/*")
	ok = len(items) == 0
	return
}

func DirIsNotEmpty(dirpath string) (ok bool, err error) {
	ok, err = DirIsEmpty(dirpath)
	return !ok, err
}

func IsRegularFile(name string) (ok bool, err error) {
	info, err := os.Lstat(name)
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

// UnlockFile 把檔案设为可读写.
func UnlockFile(name string) error {
	return os.Chmod(name, NormalFilePerm)
}

// LockFile 把檔案设为只读权限 (不可写)
func LockFile(name string) error {
	return os.Chmod(name, ReadonlyFilePerm)
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
func CopyFile(dstPath, srcPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err1 := io.Copy(dst, src)
	err2 := dst.Sync()
	return WrapErrors(err1, err2)
}

func CopyAndLockFile(dstPath, srcPath string) error {
	if err := CopyFile(dstPath, srcPath); err != nil {
		return err
	}
	return LockFile(dstPath)
}

func CopyAndUnlockFile(dstPath, srcPath string) error {
	if err := CopyFile(dstPath, srcPath); err != nil {
		return err
	}
	return UnlockFile(dstPath)
}

func DeleteFiles(files []string) (err error) {
	for _, file := range files {
		e := os.Remove(file)
		err = WrapErrors(err, e)
	}
	return err
}

// CheckFileName 尝试创建文件, 以确保文件名合法.
func CheckFileName(tempFile string) error {
	data := []byte("aabbcc") // 随便写入一些内容
	if err := WriteFile(tempFile, data, 0); err != nil {
		return err
	}
	// 如果正常创建文件, 则删除文件.
	return os.Remove(tempFile)
}

// CheckTime 检查时间字符串是否符合指定格式.
func CheckTime(layout, value string) error {
	_, err := time.Parse(layout, value)
	return err
}

// OneWaySyncDir 单向同步资料夹.
// Bug: 如果子目录名称不相同, 有可能出错.
func OneWaySyncDir(srcDir, dstDir string) error {
	err := filepath.WalkDir(srcDir, func(srcFile string, entry fs.DirEntry, err error) error {
		if err != nil {
			err2 := fmt.Errorf("error in WalkDir")
			return WrapErrors(err2, err)
		}

		// 跳过资料夹
		if entry.IsDir() {
			return nil
		}

		dstFile := filepath.Join(dstDir, entry.Name())
		_, err = os.Lstat(dstFile)
		dstNotExist := os.IsNotExist(err)
		if dstNotExist {
			err = nil
		}
		if err != nil {
			return err
		}

		// 新增文档
		if dstNotExist {
			return CopyFile(dstFile, srcFile)
		}

		// 对比文档, 覆盖文档
		srcSum, e1 := FileSum512(srcFile)
		dstSum, e2 := FileSum512(dstFile)
		if err := WrapErrors(e1, e2); err != nil {
			return err
		}
		if srcSum != dstSum {
			return CopyFile(dstFile, srcFile)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// 删除文件
	err = filepath.WalkDir(dstDir, func(dstFile string, dstEntry fs.DirEntry, err error) error {
		if err != nil {
			err2 := fmt.Errorf("error in WalkDir")
			return WrapErrors(err2, err)
		}

		if dstEntry.IsDir() {
			return nil
		}

		srcFile := filepath.Join(srcDir, dstEntry.Name())
		_, err = os.Lstat(srcFile)
		srcNotExist := os.IsNotExist(err)
		if srcNotExist {
			err = nil
		}
		if err != nil {
			return err
		}

		if srcNotExist {
			return os.Remove(dstFile)
		}
		return nil
	})
	return err
}
