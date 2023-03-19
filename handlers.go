package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/util"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/samber/lo"
)

const OK = http.StatusOK

var validate = validator.New()

// TextMsg 用于向前端返回一个简单的文本消息。
type TextMsg struct {
	Text string `json:"text"`
}

func parseValidate(form any, c *fiber.Ctx) error {
	if err := c.BodyParser(form); err != nil {
		return err
	}
	return validate.Struct(form)
}

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectPath})
}

func checkPassword(c *fiber.Ctx) error {
	f := new(model.CheckPwdForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	_, err := db.SetAESGCM(f.Password)
	return err
}

func changePassword(c *fiber.Ctx) error {
	f := new(model.ChangePwdForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	cipherKey, err := db.ChangePassword(f.OldPassword, f.NewPassword)
	if err != nil {
		return err
	}
	ProjectConfig.CipherKey = cipherKey
	writeProjectConfig()
	return nil
}

func getAllBuckets(c *fiber.Ctx) error {
	buckets, err := db.GetAllBuckets()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}

func createBucket(c *fiber.Ctx) error {
	f := new(model.CreateBucketForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	bucket, err := db.InsertBucket(f)
	if err != nil {
		return err
	}
	createBucketFolder(f.ID)
	return c.JSON(bucket)
}

func getWaitingFiles(c *fiber.Ctx) error {
	files, err := checkAndGetWaitingFiles()
	if err != nil {
		return err
	}
	return c.JSON(files)
}

// uploadNewFiles 只上传新文件,
// 若要更新现有文件, 则使用 updateFile() 函数.
// TODO: 参考 localtags 的 addFiles() 函数, 复制文件.
func uploadNewFiles(c *fiber.Ctx) error {
	f := new(model.UploadToBucketForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	files, err := checkAndGetWaitingFiles()
	if err != nil {
		return err
	}
	files = setBucketID(f.BucketID, files)
	return db.InsertFiles(files)
}

func setBucketID(bucketID string, files []*File) []*File {
	for _, file := range files {
		file.BucketID = bucketID
	}
	return files
}

func checkAndGetWaitingFiles() ([]*File, error) {
	files, err := util.GetRegularFiles(WaitingFolder)
	if err != nil {
		return nil, err
	}
	waitingFiles, err := toWaitingFiles(files)
	if err != nil {
		return nil, err
	}
	waitingFilesList := lo.Values(waitingFiles)
	err = db.CheckSameFiles(waitingFilesList)
	return waitingFilesList, err
}

func toWaitingFiles(files []string) (map[string]*File, error) {
	waitingFiles := make(map[string]*File)
	filenames := []string{}

	for _, filePath := range files {
		file, err := model.NewWaitingFile(filePath)
		if err != nil {
			return nil, err
		}

		filename := strings.ToLower(file.Name)
		if lo.Contains(filenames, filename) {
			err = fmt.Errorf(
				"發現同名檔案 (檔案名稱不分大小寫): %s", filename)
			return nil, err
		}
		filenames = append(filenames, filename)

		if same, ok := waitingFiles[file.Checksum]; ok {
			err = fmt.Errorf(
				"[%s] 與 [%s] 重複 (兩個檔案內容完全相同)", file.Name, same.Name)
			return nil, err
		}
		waitingFiles[file.Checksum] = file
	}

	return waitingFiles, nil
}
