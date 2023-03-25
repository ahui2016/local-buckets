package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

// sleep is a middleware.
func sleep(c *fiber.Ctx) error {
	time.Sleep(time.Second)
	return c.Next()
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
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	_, err := db.SetAESGCM(form.Text) // password = form.Text
	return err
}

func changePassword(c *fiber.Ctx) error {
	form := new(model.ChangePwdForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	cipherKey, err := db.ChangePassword(form.OldPassword, form.NewPassword)
	if err != nil {
		return err
	}
	ProjectConfig.CipherKey = cipherKey
	writeProjectConfig()
	return nil
}

// TODO: 输入密码后才包含加密仓库
func getAllBuckets(c *fiber.Ctx) error {
	buckets, err := db.GetAllBuckets()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}

func createBucket(c *fiber.Ctx) error {
	form := new(model.CreateBucketForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bucket, err := db.InsertBucket(form)
	if err != nil {
		return err
	}
	createBucketFolder(form.ID)
	return c.JSON(bucket)
}

func getWaitingFolder(c *fiber.Ctx) error {
	return c.JSON(TextMsg{WaitingFolder})
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

func renameWaitingFile(c *fiber.Ctx) error {
	form := new(model.RenameWaitingFileForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	exists, err := waitingFileNameExists(form.NewName)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("重命名失敗, waiting 中已有同名檔案: %s", form.NewName)
	}
	if err := db.CheckSameFilename(form.NewName); err != nil {
		return err
	}
	oldPath := filepath.Join(WaitingFolder, form.OldName)
	newPath := filepath.Join(WaitingFolder, form.NewName)
	return os.Rename(oldPath, newPath)
}

func waitingFileNameExists(name string) (ok bool, err error) {
	files, err := util.GetRegularFiles(WaitingFolder)
	if err != nil {
		return
	}
	filenames := lo.Map(files, func(filename string, _ int) string {
		return strings.ToLower(filepath.Base(filename))
	})
	ok = lo.Contains(filenames, strings.ToLower(name))
	return
}

func overwriteFile(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}

	waitingFile := new(model.MovedFile)
	waitingFile.Src = filepath.Join(WaitingFolder, form.Text) // filename = form.Text

	// 这个 file 主要是为了获取新文件的 checksum, size 等数据.
	file, err := model.NewWaitingFile(waitingFile.Src)
	if err != nil {
		return err
	}
	if err := db.CheckSameChecksum(file); err != nil {
		return err
	}

	// 这个 fileInDB 主要是为了获取 File.ID 和 BucketID.
	fileInDB, err := db.GetFileByName(file.Name)
	if err != nil {
		return err
	}
	file.ID = fileInDB.ID
	file.BucketID = fileInDB.BucketID

	waitingFile.Dst = filepath.Join(BucketsFolder, file.BucketID, file.Name)

	// 以上是收集信息及检查错误
	// 以下开始操作文件和数据库

	// tempFile 把旧文件临时移动到安全的地方
	// 在文件名区分大小写的系统里, 要注意 file.Name 与 fileInDB.Name 可能不同.
	tempFile := model.MovedFile{
		Src: filepath.Join(BucketsFolder, file.BucketID, fileInDB.Name),
		Dst: filepath.Join(TempFolder, fileInDB.Name),
	}
	if err = tempFile.Move(); err != nil {
		return err
	}

	// 移动新文件进仓库, 如果出错, 必须把旧文件移回原位.
	if err = waitingFile.Move(); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}

	// 更新数据库信息, 如果出错, 要把 waitingFile 和 tempFile 都移回原位.
	if err := db.UpdateFileContent(file); err != nil {
		err2 := waitingFile.Rollback()
		err3 := tempFile.Rollback()
		return util.WrapErrors(err, err2, err3)
	}

	// 最后删除 tempFile.
	return os.Remove(tempFile.Dst)
}

// uploadNewFiles 只上传新檔案,
// 若要更新现有檔案, 则使用 overwriteFile() 函数.
func uploadNewFiles(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	err1 := parseValidate(form, c)
	bucket, err2 := db.GetBucket(form.Text) // bucketID = form.Text
	count, err3 := db.GetInt1(stmt.CountFilesInBucket, form.Text)
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

	files = setBucketID(bucket.ID, files)
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
		Dst: filepath.Join(BucketsFolder, file.BucketID, file.Name),
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

func getRecentFiles(c *fiber.Ctx) error {
	files, err := db.GetRecentFiles()
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func getFileByID(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	file, err := db.GetFileByID(form.ID)
	if err != nil {
		return err
	}
	file.Checksum = ""
	return c.JSON(file)
}

// TODO: 在加密仓库与公开仓库之间移动文件
func moveFileToBucket(c *fiber.Ctx) error {
	form := new(model.MoveFileToBucketForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	file, err := db.GetFileByID(form.FileID)
	if err != nil {
		return err
	}
	moved := MovedFile{
		Src: filepath.Join(BucketsFolder, file.BucketID, file.Name),
		Dst: filepath.Join(BucketsFolder, form.BucketID, file.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if err := db.MoveFileToBucket(form.FileID, form.BucketID); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

func updateFileInfo(c *fiber.Ctx) error {
	form := new(model.UpdateFileInfoForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	if err := checkFileName(form.Name); err != nil {
		return err
	}
	file, err := db.GetFileByID(form.ID)
	if err != nil {
		return err
	}
	if form.Name == file.Name &&
		form.Notes == file.Notes &&
		form.Keywords == file.Keywords &&
		form.Like == file.Like &&
		form.CTime == file.CTime &&
		form.UTime == file.UTime {
		return fmt.Errorf("nothing changes (沒有變更)")
	}

	if form.UTime == file.UTime {
		form.UTime = model.Now()
	}

	moved := new(MovedFile)
	if form.Name != file.Name {
		if err := db.CheckSameFilename(form.Name); err != nil {
			return err
		}
		moved.Src = filepath.Join(BucketsFolder, file.BucketID, file.Name)
		moved.Dst = filepath.Join(BucketsFolder, file.BucketID, form.Name)
		if err := moved.Move(); err != nil {
			return err
		}
		file.Rename(form.Name)
	}

	file.Notes = form.Notes
	file.Keywords = form.Keywords
	file.Like = form.Like
	file.CTime = form.CTime
	file.UTime = form.UTime

	if err := db.UpdateFileInfo(&file); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

func checkFileName(name string) error {
	return util.CheckFileName(filepath.Join(TempFolder, name))
}

func createBackupProject(backupRoot string) error {
	notEmpty, err := util.DirIsNotEmpty(backupRoot)
	if err != nil {
		return err
	}
	if notEmpty {
		return fmt.Errorf("不是空資料夾: %s", backupRoot)
	}
	bkProjCfg := *ProjectConfig
	bkProjCfg.IsBackup = true
	bkProjCfgPath := filepath.Join(backupRoot, ProjectTOML)
	if err := util.WriteTOML(bkProjCfg, bkProjCfgPath); err != nil {
		return err
	}
	exePath := util.GetExePath()
	exeName := filepath.Base(exePath)
	bkProjExePath := filepath.Join(backupRoot, exeName)
	if err := util.CopyFile(bkProjExePath, exePath); err != nil {
		return err
	}
	bkProjBucketsDir := filepath.Join(backupRoot, BucketsFolderName)
	return util.Mkdir(bkProjBucketsDir)
}
