package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ahui2016/local-buckets/database"
	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/ahui2016/local-buckets/thumb"
	"github.com/ahui2016/local-buckets/util"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/ricochet2200/go-disk-usage/du"
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
	time.Sleep(time.Millisecond * time.Duration(ProjectConfig.ApiDelay))
	return c.Next()
}

// requireAdmin is a middleware
func requireAdmin(c *fiber.Ctx) error {
	if !db.IsLoggedIn() {
		return fmt.Errorf("該操作需要管理員權限")
	}
	return c.Next()
}

// notAllowInBackup is a middleware.
// 使用 notAllowInBackup, 表示拒絕在備份專案中使用.
func notAllowInBackup(c *fiber.Ctx) error {
	if ProjectConfig.IsBackup {
		return fmt.Errorf("這是備份專案, 不可使用該功能")
	}
	return c.Next()
}

// noCache is a middleware.
// Cache-Control: no-store will refrain from caching.
// You will always get the up-to-date response.
func noCache(c *fiber.Ctx) error {
	c.Set("Cache-Control", "no-store")
	return c.Next()
}

// 如果处理加密文档或加密仓库, 则需要管理员权限.
func checkRequireAdmin(encrypted bool) error {
	if encrypted && !db.IsLoggedIn() {
		return fmt.Errorf("處理加密檔案需要管理員權限")
	}
	return nil
}

// 如果是加密仓库, 则需要管理员权限.
func checkBucketRequireAdmin(id int64) error {
	bucket, err := db.GetBucket(id)
	if err != nil {
		return err
	}
	return checkRequireAdmin(bucket.Encrypted)
}

func parseValidate(form any, c *fiber.Ctx) error {
	if err := c.BodyParser(form); err != nil {
		return err
	}
	return validate.Struct(form)
}

func paramParseValidate(form any, c *fiber.Ctx) error {
	if err := c.ParamsParser(form); err != nil {
		return err
	}
	return validate.Struct(form)
}

func getProjectStatus(c *fiber.Ctx) error {
	projStat, err := db.GetProjStat(ProjectConfig)
	if err != nil {
		return err
	}
	return c.JSON(projStat)
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
	return writeProjectConfig()
}

func adminLogin(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	_, err := db.SetAESGCM(form.Text) // password = form.Text
	return err
}

func logoutHandler(c *fiber.Ctx) error {
	db.Logout()
	return nil
}

func getLoginStatus(c *fiber.Ctx) error {
	status := model.OneTextForm{Text: "logged-out"}
	if db.IsLoggedIn() {
		status.Text = "logged-in"
	}
	return c.JSON(status)
}

func autoGetKeywords(c *fiber.Ctx) error {
	// 等遇到性能问题再改为手动刷新吧, 数据库有索引应该效率足够高了, 不会浪费计算资源.
	keywords, err := db.AutoGetKeywords()
	if err != nil {
		return err
	}
	return c.JSON(keywords)
}

func autoGetBuckets(c *fiber.Ctx) error {
	buckets, err := db.AllBucketsStatus()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}

func getBucketHandler(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bucket, err := db.GetBucket(form.ID)
	if err != nil {
		return err
	}
	if err = checkRequireAdmin(bucket.Encrypted); err != nil {
		return err
	}
	return c.JSON(bucket)
}

func updateBucketHandler(c *fiber.Ctx) error {
	form := new(Bucket)
	if err := c.BodyParser(form); err != nil {
		return err
	}
	bucket, err := db.GetBucket(form.ID)
	if err != nil {
		return err
	}
	if err = checkRequireAdmin(bucket.Encrypted); err != nil {
		return err
	}
	if form.Name == "" {
		return fmt.Errorf(`require "Name"`)
	}
	if err = form.CheckName(); err != nil {
		return err
	}
	if form.Title == "" {
		form.Title = form.Name
	}
	if strings.EqualFold(form.Name, bucket.Name) &&
		strings.EqualFold(form.Title, bucket.Title) &&
		strings.EqualFold(form.Subtitle, bucket.Subtitle) {
		return fmt.Errorf("nothing changes (沒有變更)")
	}
	oldBucketPath := filepath.Join(BucketsFolder, bucket.Name)
	newBucketPath := filepath.Join(BucketsFolder, form.Name)
	bucketNameChanged := !strings.EqualFold(form.Name, bucket.Name)
	if bucketNameChanged {
		if err = os.Rename(oldBucketPath, newBucketPath); err != nil {
			return err
		}
	}
	err = db.UpdateBucketInfo(form)
	if err != nil && bucketNameChanged {
		err2 := os.Rename(newBucketPath, oldBucketPath)
		err = util.WrapErrors(err, err2)
	}
	return err
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
	createBucketFolder(form.Name)
	return c.JSON(bucket)
}

func deleteBucket(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bucket, err := db.GetBucket(form.ID)
	if err != nil {
		return err
	}
	if err := db.DeleteBucket(bucket.ID); err != nil {
		return err
	}
	return os.Remove(filepath.Join(BucketsFolder, bucket.Name))
}

func getWaitingFolder(c *fiber.Ctx) error {
	return c.JSON(TextMsg{WaitingFolder})
}

func getImportedFilesHandler(c *fiber.Ctx) error {
	importedFiles, err := checkGetImportedFiles()
	if e, ok := err.(model.ErrSameNameFiles); ok {
		return c.Status(400).JSON(e)
	}
	if err != nil {
		return err
	}
	return c.JSON(importedFiles)
}

func checkGetImportedFiles() (importedFiles []*File, err error) {
	files, err := util.GetRegularFiles(WaitingFolder)
	if err != nil {
		return
	}
	for _, filePath := range files {
		tomlFile := filePath + DotTOML
		if lo.Contains(files, tomlFile) {
			file, err := model.NewWaitingFile(filePath)
			if err != nil {
				return nil, err
			}
			importedFiles = append(importedFiles, file)
		}
	}
	err = db.CheckSameFiles(importedFiles)
	return
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
	if err := os.Rename(oldPath, newPath); err != nil {
		return err
	}
	return renameWaitingTOML(oldPath, newPath)
}

// oldFilePath 和 newFilePath 是指文档本身的路径, 不是 toml 的路径.
func renameWaitingTOML(oldFilePath, newFilePath string) error {
	oldTOML := oldFilePath + DotTOML
	newTOML := newFilePath + DotTOML
	if util.PathNotExists(oldTOML) {
		return nil
	}
	return os.Rename(oldTOML, newTOML)
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

	waitingFile := new(MovedFile)
	waitingFile.Src = filepath.Join(WaitingFolder, form.Text) // filename = form.Text

	// 这个 file 主要是为了获取新文档的 checksum, size 等数据.
	file, err := model.NewWaitingFile(waitingFile.Src)
	if err != nil {
		return err
	}
	if err := db.CheckSameChecksum(file); err != nil {
		return err
	}

	dbFile, err := db.GetFilePlusByName(file.Name)
	if err != nil {
		return err
	}
	if err := checkRequireAdmin(dbFile.Encrypted); err != nil {
		return err
	}

	// 如果有同名 toml, 則以 toml 的信息為準.
	// 但是, 注意, BucketName 以 dbFile 為準.
	tomlPath := waitingFile.Src + DotTOML
	if util.PathExists(tomlPath) {
		tomlFile, err := model.ImportFileFrom(tomlPath)
		if err != nil {
			return err
		}
		file.ImportFrom(tomlFile)
	}

	file.ID = dbFile.ID
	file.BucketName = dbFile.BucketName
	file.UTime = model.Now()
	waitingFile.Dst = filepath.Join(BucketsFolder, file.BucketName, file.Name)

	// 以上是收集信息及检查错误
	// 以下开始操作文档和数据库

	// tempFile 把旧文档临时移动到安全的地方
	// 在文档名区分大小写的系统里, 要注意 file.Name 与 dbFile.Name 可能不同.
	tempFile := MovedFile{
		Src: filepath.Join(BucketsFolder, file.BucketName, dbFile.Name),
		Dst: filepath.Join(TempFolder, dbFile.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}

	if dbFile.Encrypted {
		err = overwritePrivate(waitingFile, &tempFile, file)
	} else {
		err = overwritePublic(waitingFile, &tempFile, file)
	}
	return err
}

func overwritePrivate(waitingFile, tempFile *MovedFile, file *File) error {
	// 加密文档, 如果出错, 必须把旧文档移回原位.
	if err := db.EncryptFile(waitingFile.Src, waitingFile.Dst, util.ReadonlyFilePerm); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}

	// 获取加密后的 checksum
	checksum, err := util.FileSum512(waitingFile.Dst)
	if err != nil {
		return err
	}
	file.Checksum = checksum

	// 更新数据库信息, 如果出错, 要删除刚才的加密文档, 并把 tempFile 都移回原位.
	if err := db.UpdateFileContent(file); err != nil {
		err2 := os.Remove(waitingFile.Dst)
		err3 := tempFile.Rollback()
		return util.WrapErrors(err, err2, err3)
	}

	// 重新生成缩略图, 然后删除 waitingFile 和 tempFile
	createThumb(waitingFile.Src, file)
	e1 := os.Remove(waitingFile.Src)
	e2 := os.Remove(tempFile.Dst)
	return util.WrapErrors(e1, e2)
}

func overwritePublic(waitingFile, tempFile *MovedFile, file *File) error {
	// 移动新文档进仓库, 如果出错, 必须把旧文档移回原位.
	if err := waitingFile.Move(); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}
	// 更新数据库信息, 如果出错, 要把 waitingFile 和 tempFile 都移回原位.
	if err := db.UpdateFileContent(file); err != nil {
		err2 := waitingFile.Rollback()
		err3 := tempFile.Rollback()
		return util.WrapErrors(err, err2, err3)
	}
	// 重新生成缩略图, 然后删除 tempFile
	createThumb(waitingFile.Dst, file)
	return os.Remove(tempFile.Dst)
}

func downloadSmallPic(c *fiber.Ctx) error {
	file, err := checkAndGetFilePlus(c)
	if err != nil {
		return err
	}
	if !file.IsImage() {
		return fmt.Errorf("not an image (不是圖片)")
	}
	img, err := readImage(file)
	if err != nil {
		return err
	}
	dst := filepath.Join(WaitingFolder, file.Name)
	return thumb.ResizeToFile(dst, img, 0, 0)
}

func checkAndGetFilePlus(c *fiber.Ctx) (file FilePlus, err error) {
	form := new(model.FileIdForm)
	err1 := parseValidate(form, c)
	file, err2 := db.GetFilePlus(form.ID)
	if err = util.WrapErrors(err1, err2); err != nil {
		return
	}
	err = checkRequireAdmin(file.Encrypted)
	return
}

func downloadFile(c *fiber.Ctx) error {
	file, err := checkAndGetFilePlus(c)
	if err != nil {
		return err
	}
	srcPath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	dstPath := filepath.Join(WaitingFolder, file.Name)
	if util.PathExists(dstPath) {
		return fmt.Errorf("file exists: %s", dstPath)
	}
	if file.Encrypted {
		err = db.DecryptSaveFile(srcPath, dstPath, util.NormalFilePerm)
	} else {
		err = util.CopyAndUnlockFile(dstPath, srcPath)
	}
	if err != nil {
		return err
	}
	if ProjectConfig.DownloadExport {
		exported := model.ExportFileFrom(file.File)
		exportedPath := filepath.Join(WaitingFolder, file.Name+DotTOML)
		return util.WriteTOML(exported, exportedPath)
	}
	return nil
}

func setExportHandler(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	ProjectConfig.DownloadExport = form.Text == "true"
	return c.JSON(ProjectConfig.DownloadExport)
}

func importFiles(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	files, err := checkGetImportedFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		// 获取一些有用变量
		tomlPath := filepath.Join(WaitingFolder, file.Name+DotTOML)
		tomlFile, err := model.ImportFileFrom(tomlPath)
		if err != nil {
			return err
		}
		// 更新文档属性, 确定 bucket
		file.ImportFrom(tomlFile)
		bucketExists, err := db.BucketExists(file.BucketName)
		if err != nil {
			return err
		}
		if !bucketExists {
			file.BucketName = form.Text
		}
		// 获取 bucket, 检查权限
		bucket, err := db.GetBucketByName(file.BucketName)
		if err != nil {
			return err
		}
		if err := checkRequireAdmin(bucket.Encrypted); err != nil {
			return err
		}
		// 正式上传文档
		if err := encryptOrMoveWaitingFile(file, bucket.Encrypted); err != nil {
			return err
		}
		// 删除同名 toml
		if err := os.Remove(tomlPath); err != nil {
			return err
		}
	}
	return nil
}

// uploadNewFiles 只上传新檔案,
// 若要更新现有檔案, 则使用 overwriteFile() 函数.
func uploadNewFiles(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	err1 := parseValidate(form, c)
	bucket, err2 := db.GetBucketByName(form.Text) // bucketName = form.Text
	if err := util.WrapErrors(err1, err2); err != nil {
		return err
	}
	if err := checkRequireAdmin(bucket.Encrypted); err != nil {
		return err
	}
	files, err := checkAndGetWaitingFiles()
	if err != nil {
		return err
	}

	// 以上是检查阶段
	// 以下是实际执行阶段

	files = setBucketName(bucket.Name, files)
	for _, file := range files {
		if err := encryptOrMoveWaitingFile(file, bucket.Encrypted); err != nil {
			return err
		}
	}
	return nil
}

func encryptOrMoveWaitingFile(file *File, encrypted bool) error {
	if encrypted {
		return encryptWaitingFileToBucket(file)
	}
	return moveWaitingFileToBucket(file)
}

func createThumb(imgPath string, file *File) {
	thumbPath := thumbFilePath(file.ID)
	if file.IsImage() {
		fmt.Println("create thumb " + thumbPath)
		if err := thumb.SmartCrop64(imgPath, thumbPath); err != nil {
			log.Println(err)
		}
	}
}

func rebuildThumbsHandler(c *fiber.Ctx) error {
	form := new(model.FileIdRangeForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	return rebuildThumbs(form.Start, form.End)
}

// 对指定范围的文档重新生成缩略图, 例如 rebuildThumbs(1, 100),
// 对从 id=1 到 id=100 之间的文档尝试生成缩略图, 包括 1 和 100.
// 自动跳过不存在的文档 或 非图片文档.
func rebuildThumbs(start, end int64) error {
	if end < start {
		end = start
	}
	for i := start; i <= end; i++ {
		file, err := db.GetFilePlus(i)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if !file.IsImage() {
			continue
		}
		img, err := readImage(file)
		if err != nil {
			return err
		}
		thumbPath := thumbFilePath(file.ID)
		fmt.Println("rebuild thumb " + thumbPath)
		if err = thumb.SmartCropBytes64(img, thumbPath); err != nil {
			log.Println(err)
		}
	}
	return nil
}

func readImage(file FilePlus) ([]byte, error) {
	filePath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	if file.Encrypted {
		return db.DecryptFile(filePath)
	}
	return os.ReadFile(filePath)
}

func encryptWaitingFileToBucket(file *File) error {
	// srcPath 是待上传的原始文档
	srcPath := filepath.Join(WaitingFolder, file.Name)
	// dstPath 是加密后保存到加密仓库中的文档
	dstPath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	// EncryptFile 读取 srcPath 的文件, 加密后保存到 dstPath.
	if err := db.EncryptFile(srcPath, dstPath, util.ReadonlyFilePerm); err != nil {
		return err
	}
	// 获取加密后的 checksum
	checksum, err := util.FileSum512(dstPath)
	if err != nil {
		return err
	}
	file.Checksum = checksum
	// 插入新文档到数据库
	if err := db.InsertFile(file); err != nil {
		// 如果数据库出错, 要删除刚才的加密文档
		err2 := os.Remove(dstPath)
		return util.WrapErrors(err, err2)
	}
	// 获取文档 ID, 生成缩略图.
	dbFile, err := db.GetFileByName(file.Name)
	if err != nil {
		return err
	}
	createThumb(srcPath, &dbFile)
	// 一切正常, 可以删除原始文档
	return os.Remove(srcPath)
}

func moveWaitingFileToBucket(file *File) error {
	movedFile := MovedFile{
		Src: filepath.Join(WaitingFolder, file.Name),
		Dst: filepath.Join(BucketsFolder, file.BucketName, file.Name),
	}
	if err := movedFile.Move(); err != nil {
		return err
	}
	if err := db.InsertFile(file); err != nil {
		err2 := movedFile.Rollback()
		return util.WrapErrors(err, err2)
	}
	dbFile, err := db.GetFileByName(file.Name)
	if err != nil {
		return err
	}
	createThumb(movedFile.Dst, &dbFile)
	return nil
}

func setBucketName(bucketName string, files []*File) []*File {
	for _, file := range files {
		file.BucketName = bucketName
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

func getFilesHandler(c *fiber.Ctx) error {
	form, files, err := parseFilesOptions(c)
	if err != nil {
		return err
	}
	if form.UTime == "" {
		form.UTime = model.Now()
	}
	if form.Sort == "" {
		form.Sort = "utime"
	}
	if form.ID > 0 {
		files, err = db.GetFilesInBucket(form.ID, form.UTime)
	} else {
		files, err = db.GetFilesLimit(form.Sort, form.UTime)
	}
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func getPicsHandler(c *fiber.Ctx) error {
	form, files, err := parseFilesOptions(c)
	if err != nil {
		return err
	}
	if form.UTime == "" {
		form.UTime = model.Now()
	}
	if form.ID > 0 {
		files, err = db.GetPicsInBucket(form.ID, form.UTime)
	} else {
		files, err = db.GetPicsLimit(form.UTime)
	}
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func parseFilesOptions(c *fiber.Ctx) (
	form *model.FilesOptions, files []*FilePlus, err error,
) {
	form = new(model.FilesOptions)
	if err = parseValidate(form, c); err != nil {
		return
	}
	var bucket Bucket
	if form.Name != "" {
		bucket, err = db.GetBucketByName(form.Name)
		if err != nil {
			return
		}
		form.ID = bucket.ID
	}
	if form.ID <= 0 {
		return
	}
	err = checkBucketRequireAdmin(form.ID)
	return
}

func searchFiles(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	files, err := db.SearchFiles(form.Text, "", ProjectConfig.RecentFilesLimit)
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func searchPics(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	files, err := db.SearchFiles(form.Text, "image", ProjectConfig.RecentFilesLimit)
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
	file, err := db.GetFilePlus(form.ID)
	if err != nil {
		return err
	}
	if err := checkRequireAdmin(file.Encrypted); err != nil {
		return err
	}
	file.Checksum = ""
	return c.JSON(file)
}

func previewFile(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	if err := paramParseValidate(form, c); err != nil {
		return err
	}
	file, err := db.GetFilePlus(form.ID)
	if err != nil {
		return err
	}
	if !file.CanBePreviewed() {
		return fmt.Errorf("can not preview file type [%s]", file.Type)
	}
	if err := checkRequireAdmin(file.Encrypted); err != nil {
		return err
	}
	setFileType(c, file)
	filePath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	if !file.Encrypted {
		return c.SendFile(filePath)
	}
	decrypted, err := db.DecryptFile(filePath)
	if err != nil {
		return err
	}
	return c.Send(decrypted)
}

func setFileType(c *fiber.Ctx, file FilePlus) {
	if file.Type == "text/md" {
		c.Type("txt", "utf-8")
		return
	}
	ext := filepath.Ext(file.Name)
	c.Type(strings.ToLower(ext))
}

func copyToTemp(srcPath, srcChecksum string, fileID int64) (dstPath string, err error) {
	dstPath = tempFilePath(fileID)
	if util.PathExists(dstPath) {
		dstChecksum, err := util.FileSum512(dstPath)
		if err != nil {
			return "", err
		}
		if dstChecksum == srcChecksum {
			return dstPath, nil
		}
	}
	err = util.CopyFile(dstPath, srcPath)
	fmt.Printf("copyToTemp write %s\n", dstPath)
	return
}

func moveFileToBucket(c *fiber.Ctx) (err error) {
	form := new(model.MoveFileToBucketForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	file, e1 := db.GetFilePlus(form.FileID)
	bucket, e2 := db.GetBucketByName(form.BucketName)
	if err := util.WrapErrors(e1, e2); err != nil {
		return err
	}

	// 先处理在公开仓库与加密仓库之间移动文档的情况 (需要加密或解密)
	if file.Encrypted != bucket.Encrypted {
		if !db.IsLoggedIn() {
			return fmt.Errorf("檔案移進或移出加密倉庫需要管理員權限")
		}
		direction := lo.Ternary(file.Encrypted, "Pri->Pub", "Pub->Pri")
		if err = moveFileBetweenPubAndPri(file, bucket.Name, direction); err != nil {
			return err
		}
	}

	// 再处理 “公开仓库之间” 或 “加密仓库之间” 移动文档的情况 (不需要加密解密)
	if file.Encrypted == bucket.Encrypted {
		moved := MovedFile{
			Src: filepath.Join(BucketsFolder, file.BucketName, file.Name),
			Dst: filepath.Join(BucketsFolder, bucket.Name, file.Name),
		}
		if err := moved.Move(); err != nil {
			return err
		}
		if err := db.MoveFileToBucket(form.FileID, bucket.Name); err != nil {
			err2 := moved.Rollback()
			return util.WrapErrors(err, err2)
		}
	}

	// 最后获取更新后的文件, 返回给前端
	fileplus, err := db.GetFilePlus(file.ID)
	if err != nil {
		return err
	}
	return c.JSON(fileplus)
}

// direction is "Pri->Pub" or "Pub->Pri"
func moveFileBetweenPubAndPri(file FilePlus, newBucketName, direction string) (err error) {
	srcPath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	dstPath := filepath.Join(BucketsFolder, newBucketName, file.Name)

	if direction == "Pub->Pri" {
		// 从公开转到加密, 还要删除临时文档
		if err = removeTempFile(file.ID); err != nil {
			return err
		}
		err = db.EncryptFile(srcPath, dstPath, util.ReadonlyFilePerm)
	} else {
		err = db.DecryptSaveFile(srcPath, dstPath, util.ReadonlyFilePerm)
	}
	if err != nil {
		return err
	}
	// 获取新的 checksum
	checksum, err := util.FileSum512(dstPath)
	if err != nil {
		return err
	}
	if err := db.UpdateChecksumAndBucket(file.ID, checksum, newBucketName); err != nil {
		err2 := os.Remove(dstPath)
		return util.WrapErrors(err, err2)
	}
	// 一切正常, 可以删除原始文档
	return os.Remove(srcPath)
}

func updateFileInfo(c *fiber.Ctx) error {
	form := new(model.UpdateFileInfoForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	if err := checkFileName(form.Name); err != nil {
		return err
	}
	file, err := db.GetFilePlus(form.ID)
	if err != nil {
		return err
	}
	if err := checkRequireAdmin(file.Encrypted); err != nil {
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
	err1 := util.CheckTime(model.RFC3339, form.CTime)
	err2 := util.CheckTime(model.RFC3339, form.UTime)
	if err := util.WrapErrors(err1, err2); err != nil {
		return err
	}

	moved := new(MovedFile)
	if form.Name != file.Name {
		if err := db.CheckSameFilename(form.Name); err != nil {
			return err
		}
		moved.Src = filepath.Join(BucketsFolder, file.BucketName, file.Name)
		moved.Dst = filepath.Join(BucketsFolder, file.BucketName, form.Name)
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

	if err := db.UpdateFileInfo(&file.File); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	fileplus, err := db.GetFilePlus(file.ID)
	if err != nil {
		return err
	}
	return c.JSON(fileplus)
}

func checkFileName(name string) error {
	return util.CheckFileName(filepath.Join(TempFolder, name))
}

func deleteBKProjHandler(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProj := form.Text
	if lo.IndexOf(ProjectConfig.BackupProjects, bkProj) < 0 {
		return c.Status(404).SendString("not found: " + bkProj)
	}
	return deleteBKProjFromConfig(bkProj)
}

func createBKProjHandler(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProjRoot := form.Text
	if util.PathNotExists(bkProjRoot) {
		return c.Status(404).SendString("not found: " + bkProjRoot)
	}
	if err := createBackupProject(bkProjRoot); err != nil {
		return err
	}
	return addBKProjToConfig(bkProjRoot)
}

func createBackupProject(bkProjRoot string) error {
	notEmpty, err := util.DirIsNotEmpty(bkProjRoot)
	if err != nil {
		return err
	}
	if notEmpty {
		return fmt.Errorf("不是空資料夾: %s", bkProjRoot)
	}
	bkProjCfg := *ProjectConfig
	bkProjCfg.IsBackup = true
	bkProjCfg.LastBackupAt = ""
	bkProjCfgPath := filepath.Join(bkProjRoot, ProjectTOML)
	if err := util.WriteTOML(bkProjCfg, bkProjCfgPath); err != nil {
		return err
	}
	bkProjBucketsDir := filepath.Join(bkProjRoot, BucketsFolderName)
	bkProjTempDir := filepath.Join(bkProjRoot, TempFolderName)
	bkProjPublicDir := filepath.Join(bkProjRoot, PublicFolderName)
	bkProjThumbsDir := filepath.Join(bkProjPublicDir, ThumbsFolderName)
	e1 := util.MkdirIfNotExists(bkProjBucketsDir)
	e2 := util.MkdirIfNotExists(bkProjTempDir)
	e3 := util.MkdirIfNotExists(bkProjPublicDir)
	e4 := util.MkdirIfNotExists(bkProjThumbsDir)
	return util.WrapErrors(e1, e2, e3, e4)
}

func getBKProjStat(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProjRoot := form.Text
	if util.PathNotExists(bkProjRoot) {
		return c.Status(404).SendString("not found: " + bkProjRoot)
	}
	bk, bkProjStat, err := openBackupDB(bkProjRoot)
	if err != nil {
		return err
	}
	defer bk.DB.Close()
	return c.JSON(bkProjStat)
}

// 注意 open 后应立即 defer bk.DB.Close()
func openBackupDB(bkProjRoot string) (*DB, *ProjectStatus, error) {
	bkPath := filepath.Join(bkProjRoot, DatabaseFileName)
	bkProjCfgPath := filepath.Join(bkProjRoot, ProjectTOML)

	bkProjCfg, err := readProjCfgFrom(bkProjCfgPath)
	if err != nil {
		return nil, nil, err
	}

	bk, err := database.OpenDB(bkPath, &bkProjCfg)
	if err != nil {
		return nil, nil, err
	}

	bkProjStat, err := bk.GetProjStat(&bkProjCfg)
	return bk, &bkProjStat, err
}

func repairFilesHandler(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProjRoot := form.Text

	bk, _, err := getDatabaseFrom(bkProjRoot)
	if err != nil {
		return err
	}
	defer bk.DB.Close()

	// 要先同步文件夹, 否则有可能发生路径错误.
	bkBucketsDir := filepath.Join(bkProjRoot, BucketsFolderName)
	if err := syncBuckets(bkBucketsDir, bk); err != nil {
		return err
	}

	return repairDamagedFiles(bkProjRoot, bk)
}

func checkNow(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}

	root := form.Text
	db1, cfg, err := getDatabaseFrom(root)
	if err != nil {
		return err
	}
	if db1.IsBackup {
		defer db1.DB.Close()
	}

	if err = checkFilesChecksum(root, db1); err != nil {
		return err
	}

	projStat, err := db1.GetProjStat(cfg)
	if err != nil {
		return err
	}
	return c.JSON(projStat)
}

func getDatabaseFrom(root string) (*DB, *Project, error) {
	yes, err := util.SamePath(root, ProjectRoot)
	if err != nil {
		return nil, nil, err
	}
	if yes {
		return db, ProjectConfig, nil
	}
	bk, bkStat, err := openBackupDB(root)
	return bk, bkStat.Project, err
}

// root 是被檢查的專案根目錄, db1 是被檢查的專案數據庫.
func checkFilesChecksum(root string, db1 *DB) error {
	var totalChecked int64
	files, err := db1.GetFilesNeedCheck(ProjectConfig.CheckInterval * Day)
	if err != nil {
		return err
	}
	for _, file := range files {
		// 先檢查一個檔案
		if _, err = checkFile(root, file, db1); err != nil {
			return err
		}
		// 然後根據 ProjectConfig.CheckSizeLimit 終止檢查
		totalChecked += file.Size
		if totalChecked > ProjectConfig.CheckSizeLimit*GB {
			return nil
		}
	}
	return nil
}

func checkFile(root string, file *File, db1 *DB) (damaged bool, err error) {
	filePath := filepath.Join(root, BucketsFolderName, file.BucketName, file.Name)
	sum, err := util.FileSum512(filePath)
	if err != nil {
		return
	}
	if sum != file.Checksum {
		file.Damaged = true
	}
	file.Checked = model.Now()
	err = db1.SetFileCheckedDamaged(file)
	return file.Damaged, err
}

func recheckFile(db1 *DB, root string, fileID int64) (damaged bool, err error) {
	// 如果在 db1 中标记了该文件已损坏，则直接返回结果。
	file, err := db1.GetFileByID(fileID)
	if err != nil {
		return
	}
	if file.Damaged {
		return true, nil
	}

	// 如果在 db1 中标记了该文件未损坏，则再检查一次。
	return checkFile(root, &file, db1)
}

// 从 badDB 中找出 badFiles, 然后尝试从 goodDB 中获取未损坏版本进行修复。
// 如果修复成功，则将 badFile 标记为未损坏，
// 如果不可修复，则不进行任何操作, badFile 仍然保持 "已损坏" 的标记。
func repair(badDB, goodDB *DB, badRoot, goodRoot string) error {
	badFiles, err := badDB.GetDamagedFiles()
	if err != nil {
		return err
	}
	for _, file := range badFiles {
		// 如果 goodDB 中的文件已损坏或找不到文件，则无法修复，如果未损坏则进行修复。
		damaged, err := recheckFile(goodDB, goodRoot, file.ID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return err
		}
		if damaged {
			continue
		}

		// 进行修复
		goodFilePath := filepath.Join(goodRoot, BucketsFolderName, file.BucketName, file.Name)
		badFilePath := filepath.Join(badRoot, BucketsFolderName, file.BucketName, file.Name)
		if err := util.CopyFile(badFilePath, goodFilePath); err != nil {
			return err
		}

		// 更新校验日期，标记为未损坏
		file.Damaged = false
		file.Checked = model.Now()

		if err = badDB.SetFileCheckedDamaged(&file.File); err != nil {
			return err
		}
	}
	return nil
}

func damagedInProjects(db1, db2 *DB) (int64, error) {
	info1, e1 := db1.GetProjStat(ProjectConfig)
	info2, e2 := db2.GetProjStat(ProjectConfig)
	return info1.DamagedCount + info2.DamagedCount, util.WrapErrors(e1, e2)
}

// repairDamagedFiles 自动修复文件，对于主仓库里的损坏文件，尝试从备份仓库中获取未损坏版本，
// 对于备份仓库中的损坏文件则尝试从主仓库中获取未损坏版本，如果一个文件在主仓库及备份仓库中都损坏了，
// 则无法修复该文件，后续提醒用户手动修复。
func repairDamagedFiles(bkProjRoot string, bk *DB) error {
	if err := repair(db, bk, ProjectRoot, bkProjRoot); err != nil {
		return err
	}
	if err := repair(bk, db, bkProjRoot, ProjectRoot); err != nil {
		return err
	}

	// 经自动修复后，再次检查有没有损坏文件。
	n, err := damagedInProjects(bk, db)
	if err != nil {
		return err
	}
	if n > 0 {
		return fmt.Errorf("仍有 %d 個受損檔案未修復, 請手動修復", n)
	}
	return nil
}

func syncBackup(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProjStat, err := syncToBackupProject(form.Text)
	if err != nil {
		return err
	}
	e1 := projCfgUpdateAndSync(bkProjStat)
	e2 := syncPublicFolder(form.Text)
	e3 := syncExeFile(bkProjStat.Root)
	return util.WrapErrors(e1, e2, e3)
}

func syncPublicFolder(bkProjRoot string) error {
	bkPublicFolder := filepath.Join(bkProjRoot, PublicFolderName)
	bkThumbsFolder := filepath.Join(bkPublicFolder, ThumbsFolderName)
	e1 := util.OneWaySyncDir(PublicFolder, bkPublicFolder)
	e2 := util.OneWaySyncDir(ThumbsFolder, bkThumbsFolder)
	return util.WrapErrors(e1, e2)
}

func syncExeFile(bkProjRoot string) error {
	exePath := util.GetExePath()
	exeName := filepath.Base(exePath)
	bkExePath := filepath.Join(bkProjRoot, exeName)
	if util.PathNotExists(bkExePath) {
		return util.CopyFile(bkExePath, exePath)
	}

	exeSum, e1 := util.FileSum512(exePath)
	bkExeSum, e2 := util.FileSum512(bkExePath)
	if err := util.WrapErrors(e1, e2); err != nil {
		return err
	}
	if exeSum == bkExeSum {
		return nil
	}
	return util.CopyFile(bkExePath, exePath)
}

// syncToBackupProject 以源仓库为准单向同步，
// 最终效果相当于清空备份仓库后把主仓库的全部文档复制到备份仓库。
// 注意这里不能使用事务 TX, 因为一旦回滚, 批量恢复文档名称太麻烦了.
func syncToBackupProject(bkProjRoot string) (*ProjectStatus, error) {
	projStat, err := db.GetProjStat(ProjectConfig)
	if err != nil {
		return nil, err
	}
	bk, bkProjStat, err := openBackupDB(bkProjRoot)
	if err != nil {
		return nil, err
	}
	defer bk.DB.Close()

	if bkProjStat, err = syncProjectConfig(bkProjStat); err != nil {
		return nil, err
	}

	if err := checkBackupDiskUsage(bkProjRoot, bkProjStat, &projStat); err != nil {
		return nil, err
	}

	bkBucketsDir := filepath.Join(bkProjRoot, BucketsFolderName)
	bkTemp := filepath.Join(bkProjRoot, TempFolderName)

	// 先处理仓库, 包括新建或删除仓库资料夹.
	if err := syncBuckets(bkBucketsDir, bk); err != nil {
		return nil, err
	}

	// 获取发生了变化的文档
	changedFiles, err := getChangedFiles(db, bk, bkBucketsDir, bkTemp)
	if err != nil {
		return nil, err
	}
	// 同步文档(单向同步)
	if err = changedFiles.Sync(); err != nil {
		return nil, err
	}
	return bkProjStat, nil
}

type ChangedFiles struct {
	DB         *DB
	BK         *DB
	BKBuckets  string
	BKTemp     string
	Deleted    []int64
	Updated    []int64
	Moved      []int64
	Overwrited []int64
	Inserted   []int64
}

func getChangedFiles(db, bk *DB, bkBuckets, bkTemp string) (files ChangedFiles, err error) {
	var (
		bkFile File
		dbFile File
	)
	files.DB = db
	files.BK = bk
	files.BKBuckets = bkBuckets
	files.BKTemp = bkTemp

	rows, err := bk.Query(stmt.GetAllFiles)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		bkFile, err = database.ScanFile(rows)
		if err != nil {
			return
		}
		dbFile, err = db.GetFileByID(bkFile.ID)

		// 已被删除的文档
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
			files.Deleted = append(files.Deleted, bkFile.ID)
			continue
		}
		if err != nil {
			return
		}

		// 更新了除 Checksum 和 BucketName 以外的属性的文档
		if !filesHaveSameProperties(bkFile, dbFile) {
			files.Updated = append(files.Updated, bkFile.ID)
		}

		// 已被移动 (到另一个仓库) 的文档
		if bkFile.BucketName != dbFile.BucketName {
			files.Moved = append(files.Moved, bkFile.ID)
		}

		// 更新了内容 (Checksum 已改变) 的文档
		if bkFile.Checksum != dbFile.Checksum {
			files.Overwrited = append(files.Overwrited, bkFile.ID)
		}
	}
	if err = rows.Err(); err != nil {
		return
	}
	rows.Close()

	// 以下代码获取新增的文档
	rows, err = db.Query(stmt.GetAllFiles)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		dbFile, err = database.ScanFile(rows)
		if err != nil {
			return
		}
		_, err = bk.GetFileByID(dbFile.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
			files.Inserted = append(files.Inserted, dbFile.ID)
		}
		if err != nil {
			return
		}
	}
	return files, rows.Err()
}

func (files ChangedFiles) getFilePair(id int64) (bkFile, dbFile FilePlus, err error) {
	bkFile, e1 := files.BK.GetFilePlus(id)
	dbFile, e2 := files.DB.GetFilePlus(id)
	err = util.WrapErrors(e1, e2)
	return
}

func (files ChangedFiles) Sync() (err error) {
	// 这里几种操作的顺序不能错, 比如最好是最后才添加文档.
	if err = files.syncDelete(); err != nil {
		fmt.Println("delete", err)
		return
	}
	if err = files.syncOverwrite(); err != nil {
		fmt.Println("overwrite", err)
		return
	}
	if err = files.syncMove(); err != nil {
		fmt.Println("move", err)
		return
	}
	if err = files.syncUpdate(); err != nil {
		fmt.Println("upddate", err)
		return
	}
	if err = files.syncInsert(); err != nil {
		fmt.Println("insert", err)
		return
	}
	return nil
}

func (files ChangedFiles) syncDelete() error {
	for _, id := range files.Deleted {
		f, err := files.BK.GetFileByID(id)
		if err != nil {
			return err
		}
		if err := files.BK.DeleteFile(files.BKBuckets, files.BKTemp, thumbFilePath(id), &f); err != nil {
			return err
		}
	}
	return nil
}

func (files ChangedFiles) syncUpdate() error {
	for _, id := range files.Updated {
		bkFile, dbFile, err := files.getFilePair(id)
		if err != nil {
			return err
		}
		if err = updateBKFile(&bkFile, &dbFile, files.BK, files.BKBuckets); err != nil {
			return err
		}
	}
	return nil
}

func (files ChangedFiles) syncMove() error {
	for _, id := range files.Moved {
		bkFile, dbFile, err := files.getFilePair(id)
		if err != nil {
			return err
		}
		if bkFile.Encrypted != dbFile.Encrypted {
			// 需要复制文档
			err = moveEncrypedBKFile(files.BKBuckets, files.BKTemp, bkFile, dbFile, files.BK)
		} else {
			// 可以直接移动
			err = moveBKFileToBucket(files.BKBuckets, dbFile.BucketName, &bkFile, files.BK)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (files ChangedFiles) syncOverwrite() error {
	for _, id := range files.Overwrited {
		bkFile, dbFile, err := files.getFilePair(id)
		if err != nil {
			return err
		}
		if err = overwriteBKFile(files.BKBuckets, files.BKTemp, &dbFile, &bkFile, files.BK); err != nil {
			return err
		}
	}
	return nil
}

func (files ChangedFiles) syncInsert() error {
	for _, id := range files.Inserted {
		dbFile, err := files.DB.GetFileByID(id)
		if err != nil {
			return err
		}
		if err = insertBKFile(files.BKBuckets, &dbFile, files.BK); err != nil {
			return err
		}
	}
	return nil
}

func updateBKFile(bkFile, dbFile *FilePlus, bk *DB, bkBucketsDir string) error {
	// 如果文档名不一致, 还要重命名
	moved := new(MovedFile)
	if bkFile.Name != dbFile.Name {
		moved.Src = filepath.Join(bkBucketsDir, bkFile.BucketName, bkFile.Name)
		moved.Dst = filepath.Join(bkBucketsDir, bkFile.BucketName, dbFile.Name)
		if err := moved.Move(); err != nil {
			return err
		}
	}
	if err := bk.UpdateBackupFileInfo(&dbFile.File); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

// 更新备份专案的 Title, Subtitle, Cipherkey.
func syncProjectConfig(bkProjStat *ProjectStatus) (*ProjectStatus, error) {
	bkProjStat.Project.Title = ProjectConfig.Title
	bkProjStat.Project.Subtitle = ProjectConfig.Subtitle
	bkProjStat.Project.CipherKey = ProjectConfig.CipherKey

	bkProjCfgPath := filepath.Join(bkProjStat.Root, ProjectTOML)
	err := util.WriteTOML(bkProjStat.Project, bkProjCfgPath)
	return bkProjStat, err
}

// 同步仓库信息到备份仓库 (包括仓库资料夹重命名)
// 注意这里不能使用事务 TX, 因为一旦回滚, 批量恢复资料夹名称太麻烦了.
func syncBuckets(bkBucketsDir string, bk *DB) error {
	allDBBuckets, e1 := db.GetAllBuckets()
	allBKBuckets, e2 := bk.GetAllBuckets()
	if err := util.WrapErrors(e1, e2); err != nil {
		return err
	}
	for _, bkBucket := range allBKBuckets {
		bucket, err := db.GetBucket(bkBucket.ID)
		// 如果源仓库不存在, 则备份仓库也应删除.
		// Bug: 这里不删除备份仓库的资料夹, 因为此时里面可能还有文件. 留着资料夹好像问题不大, 暂时不处理这个 bug.
		if errors.Is(err, sql.ErrNoRows) || bucket.ID == 0 {
			err = nil
			if err := bk.DeleteBucket(bkBucket.ID); err != nil {
				return err
			}
			continue // 这句是必须的
		}
		if err != nil {
			return err
		}
		// 从这里开始 bucket 和 bkBucket 同时存在, 是同一个仓库的新旧版本.

		// 如果仓库资料夹名称发生了改变, 则备份仓库的资料夹也要重命名
		oldName := bkBucket.Name
		newName := bucket.Name
		if oldName != newName {
			if err := renameBucket(oldName, newName, bkBucketsDir, bucket.ID, bk); err != nil {
				return err
			}
			// 这里不能 continue
		}

		// 处理完 Name, 剩下有可能改变的就只有 Title 和 Subtitle 了.
		if bkBucket.Title != bucket.Title || bkBucket.Subtitle != bucket.Subtitle {
			if err := bk.UpdateBucketTitle(&bucket); err != nil {
				return err
			}
		}
	}

	// 新增仓库
	for _, bucket := range allDBBuckets {
		bkBucket, err := bk.GetBucket(bucket.ID)
		if errors.Is(err, sql.ErrNoRows) || bkBucket.ID == 0 {
			err = nil
			bucketPath := filepath.Join(bkBucketsDir, bucket.Name)
			if err2 := util.Mkdir(bucketPath); err2 != nil {
				return err2
			}
			if err := bk.InsertBucketWithID(bucket); err != nil {
				err2 := os.Remove(bucketPath)
				return util.WrapErrors(err, err2)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func renameBucket(oldName, newName, bkBuckets string, bucketid int64, bk *DB) error {
	oldpath := filepath.Join(bkBuckets, oldName)
	newpath := filepath.Join(bkBuckets, newName)
	if err := os.Rename(oldpath, newpath); err != nil {
		return err
	}
	if err := bk.UpdateBucketName(newName, bucketid); err != nil {
		err2 := os.Rename(newpath, oldpath)
		return util.WrapErrors(err, err2)
	}
	return nil
}

func moveBKFileToBucket(
	bkBucketsDir, newBucketName string, bkFile *FilePlus, bk *DB,
) error {
	moved := MovedFile{
		Src: filepath.Join(bkBucketsDir, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkBucketsDir, newBucketName, bkFile.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if err := bk.MoveFileToBucket(bkFile.ID, newBucketName); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

// 注意, 移动加密文档时, 不可重新加密, 以免 Checksum 发生变化.
func moveEncrypedBKFile(
	bkBuckets, bkTemp string, bkFile, dbFile FilePlus, bk *DB,
) error {
	tempFile := MovedFile{
		Src: filepath.Join(bkBuckets, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkTemp, bkFile.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}

	dstFile := filepath.Join(bkBuckets, dbFile.BucketName, dbFile.Name)
	srcFile := filepath.Join(BucketsFolder, dbFile.BucketName, dbFile.Name)
	if err := util.CopyAndLockFile(dstFile, srcFile); err != nil {
		return err
	}
	if err := bk.MoveFileToBucket(bkFile.ID, dbFile.BucketName); err != nil {
		err2 := os.Remove(dstFile)
		err3 := tempFile.Rollback()
		return util.WrapErrors(err, err2, err3)
	}
	return os.Remove(tempFile.Dst)
}

// 備份專案中的檔案 與 源專案中的檔案, 兩者的屬性相同嗎?
// 只對比一部分屬性. 在執行本函數之前, 應先同步 Checksum 和 BucketName,
// 因此在這裡不對比 Checksum, Size 和 BucketName.
// 另外, Checked 和 Damaged 也不對比.
func filesHaveSameProperties(bkFile, file File) bool {
	return file.Name == bkFile.Name &&
		file.Notes == bkFile.Notes &&
		file.Keywords == bkFile.Keywords &&
		file.Like == bkFile.Like &&
		file.CTime == bkFile.CTime &&
		file.UTime == bkFile.UTime
}

func insertBKFile(bkBuckets string, file *File, bk *DB) error {
	dstFile := filepath.Join(bkBuckets, file.BucketName, file.Name)
	srcFile := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	if err := util.CopyAndLockFile(dstFile, srcFile); err != nil {
		return err
	}
	if err := bk.InsertFileWithID(file); err != nil {
		err2 := os.Remove(dstFile)
		return util.WrapErrors(err, err2)
	}
	return nil
}

// 仅覆盖文档内容, 不改变其他任何信息.
func overwriteBKFile(bkBucketsDir, bkTemp string, dbFile, bkFile *FilePlus, bk *DB) error {
	// tempFile 把旧文档临时移动到安全的地方
	tempFile := MovedFile{
		Src: filepath.Join(bkBucketsDir, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkTemp, bkFile.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}

	// 复制新文档到备份仓库, 如果出错, 必须把旧文档移回原位.
	newFileDst := tempFile.Src
	newFileSrc := filepath.Join(BucketsFolder, dbFile.BucketName, dbFile.Name)
	if err := util.CopyAndLockFile(newFileDst, newFileSrc); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}

	// 更新数据库信息, 如果出错, 要删除 newFile 并把 tempFile 都移回原位.
	if err := bk.UpdateFileContent(&dbFile.File); err != nil {
		err2 := os.Remove(newFileDst)
		err3 := tempFile.Rollback()
		return util.WrapErrors(err, err2, err3)
	}

	// 最后删除 tempFile.
	return os.Remove(tempFile.Dst)
}

func checkBackupDiskUsage(bkProjRoot string, bkStat, projStat *ProjectStatus) error {
	usage := du.NewDiskUsage(bkProjRoot)
	addUp := projStat.TotalSize - bkStat.TotalSize // 備份後將會增加的體積
	if addUp <= 0 {
		return nil
	}
	var margin uint64 = 1 << 30 // 1GB (現在U盤也是100GB起步了)
	if uint64(addUp)+margin > usage.Available() {
		return fmt.Errorf("not enough space (備份專案空間不足)")
	}
	return nil
}

func deleteFile(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	err1 := parseValidate(form, c)
	file, err2 := db.GetFilePlus(form.ID)
	if err := util.WrapErrors(err1, err2); err != nil {
		return err
	}
	if err := checkRequireAdmin(file.Encrypted); err != nil {
		return err
	}
	if err := removeTempFile(file.ID); err != nil {
		return err
	}
	return db.DeleteFile(BucketsFolder, TempFolder, thumbFilePath(file.ID), &file.File)
}

func createNewNote(c *fiber.Ctx) error {
	f, err := os.CreateTemp(WaitingFolder, "note-*.md")
	if err != nil {
		return err
	}
	defer f.Close()
	content := `This document is encoded in UTF-8
このファイルは　UTF-8　でエンコードされています
本文檔採用 UTF-8 編碼
`
	if _, err = f.WriteString(content); err != nil {
		return err
	}
	return c.JSON(TextMsg{f.Name()})
}

func damagedFilesHandler(c *fiber.Ctx) error {
	files, err := db.GetDamagedFiles()
	if err != nil {
		return err
	}
	for _, file := range files {
		if err := checkRequireAdmin(file.Encrypted); err != nil {
			return err
		}
	}
	return c.JSON(files)
}
