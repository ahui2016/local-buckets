package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
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
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectRoot})
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
	if e, ok := err.(model.ErrSameNameFiles); ok {
		return c.Status(400).JSON(e)
	}
	if err != nil {
		return err
	}
	return c.JSON(files)
}

// uploadNewFiles 只上传新文件,
// 若要更新现有文件, 则使用 updateFile() 函数.
func uploadNewFiles(c *fiber.Ctx) error {
	f := new(model.UploadToBucketForm)
	err1 := parseValidate(f, c)
	bucket, err2 := db.GetBucket(f.BucketID)
	count, err3 := db.GetInt1(stmt.CountFilesInBucket, f.BucketID)
	if err := util.WrapErrors(err1, err2, err3); err != nil {
		return err
	}
	files, err := checkAndGetWaitingFiles()
	if err != nil {
		return err
	}
	filesLength := int64(len(files))
	if filesLength+count > bucket.Capacity {
		return fmt.Errorf(
			"待上傳檔案(%d) + 已有檔案(%d) > 倉庫容量(%d)", filesLength, count, bucket.Capacity)
	}

	// 以上是检查阶段
	// 以下是实际执行阶段

	files = setBucketID(f.BucketID, files)
	movedFiles, err := moveWaitingFiles(files)
	if err != nil {
		err2 := rollbackMovedFiles(movedFiles)
		return util.WrapErrors(err, err2)
	}
	if err := db.InsertFiles(files); err != nil {
		err2 := rollbackMovedFiles(movedFiles)
		return util.WrapErrors(err, err2)
	}
	return nil
}

func rollbackMovedFiles(movedFiles []MovedFile) (allErr error) {
	for _, file := range movedFiles {
		if err := file.Rollback(); err != nil {
			allErr = util.WrapErrors(allErr, err)
		}
	}
	return
}

func moveWaitingFiles(files []*File) (movedFiles []MovedFile, err error) {
	for _, file := range files {
		movedFile, err := moveWaitingFileToBucket(file)
		if err != nil {
			return nil, err
		}
		movedFiles = append(movedFiles, movedFile)
	}
	return
}

func moveWaitingFileToBucket(file *File) (MovedFile, error) {
	movedFile := MovedFile{
		Src: filepath.Join(WaitingFolder, file.Name),
		Dst: filepath.Join(ProjectRoot, file.BucketID, file.Name),
	}
	err := movedFile.Move()
	return movedFile, err
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
