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

func autoGetBuckets(c *fiber.Ctx) error {
	buckets, err := db.AllBucketsStatus()
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
	createBucketFolder(form.Name)
	return c.JSON(bucket)
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
	if err := db.EncryptFile(waitingFile.Src, waitingFile.Dst); err != nil {
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

func downloadFile(c *fiber.Ctx) error {
	form := new(model.FileIdForm)
	err := parseValidate(form, c)
	file, err2 := db.GetFilePlus(form.ID)
	if err := util.WrapErrors(err, err2); err != nil {
		return err
	}
	if err := checkRequireAdmin(file.Encrypted); err != nil {
		return err
	}
	srcPath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	dstPath := filepath.Join(WaitingFolder, file.Name)
	if util.PathExists(dstPath) {
		return fmt.Errorf("file exists: %s", dstPath)
	}
	if file.Encrypted {
		err = db.DecryptSaveFile(srcPath, dstPath)
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
	for i := start; i <= end; i++ {
		file, err := db.GetFilePlus(i)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		if !file.IsImage() {
			continue
		}
		filePath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
		var img []byte
		if file.Encrypted {
			img, err = db.DecryptFile(filePath)
		} else {
			img, err = os.ReadFile(filePath)
		}
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

func encryptWaitingFileToBucket(file *File) error {
	// srcPath 是待上传的原始文档
	srcPath := filepath.Join(WaitingFolder, file.Name)
	// dstPath 是加密后保存到加密仓库中的文档
	dstPath := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	// EncryptFile 读取 srcPath 的文件, 加密后保存到 dstPath.
	if err := db.EncryptFile(srcPath, dstPath); err != nil {
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

func getRecentFiles(c *fiber.Ctx) error {
	form, files, err := parseBucketIdForm(c)
	if err != nil {
		return err
	}
	if form.ID > 0 {
		files, err = db.RecentFilesInBucket(form.ID)
	} else {
		files, err = db.GetRecentFiles()
	}
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func getRecentPics(c *fiber.Ctx) error {
	form, files, err := parseBucketIdForm(c)
	if err != nil {
		return err
	}
	if form.ID > 0 {
		files, err = db.RecentPicsInBucket(form.ID)
	} else {
		files, err = db.GetRecentPics()
	}
	if err != nil {
		return err
	}
	return c.JSON(files)
}

func parseBucketIdForm(c *fiber.Ctx) (
	form *model.BucketIdForm, files []*FilePlus, err error,
) {
	form = new(model.BucketIdForm)
	e1 := parseValidate(form, c)
	if form.ID == 0 {
		return form, files, nil
	}
	e2 := checkBucketRequireAdmin(form.ID)
	err = util.WrapErrors(e1, e2)
	return
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
	file, err := db.GetFilePlusWithChecksum(form.ID)
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
		// 因为删除文档时遇到了文档被占用的错误, 试试使用临时文档看能否解决问题.
		tempFile, err := copyToTemp(filePath, file.Checksum, file.ID)
		if err != nil {
			return err
		}
		return c.SendFile(tempFile)
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

// TODO: move thumbs
// TODO: 如果是加密文档，要求管理员权限
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
		err = db.EncryptFile(srcPath, dstPath)
	} else {
		err = db.DecryptSaveFile(srcPath, dstPath)
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

// TODO: update thumbs filename
// TODO: 如果是加密文档，要求管理员权限
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

	if err := db.UpdateFileInfo(&file); err != nil {
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
func openBackupDB(bkProjRoot string) (*database.DB, *ProjectStatus, error) {
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

func getDatabaseFrom(root string) (*database.DB, *Project, error) {
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
func checkFilesChecksum(root string, db1 *database.DB) error {
	var totalChecked int64
	files, err := db1.GetFilesNeedCheck(ProjectConfig.CheckInterval * Day)
	if err != nil {
		return err
	}
	for _, file := range files {
		// 先檢查一個檔案
		if err = checkFile(root, file, db1); err != nil {
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

func checkFile(root string, file *File, db1 *database.DB) error {
	filePath := filepath.Join(root, BucketsFolderName, file.BucketName, file.Name)
	sum, err := util.FileSum512(filePath)
	if err != nil {
		return err
	}
	if sum != file.Checksum {
		file.Damaged = true
	}
	file.Checked = model.Now()
	return db1.SetFileCheckedDamaged(file)
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
	e1 := projCfgUpdateTime(bkProjStat)
	e2 := syncPublicFolder(form.Text)
	e3 := syncExec(bkProjStat.Root)
	return util.WrapErrors(e1, e2, e3)
}

func syncPublicFolder(bkProjRoot string) error {
	bkPublicFolder := filepath.Join(bkProjRoot, PublicFolderName)
	bkThumbsFolder := filepath.Join(bkPublicFolder, ThumbsFolderName)
	e1 := util.OneWaySyncDir(PublicFolder, bkPublicFolder)
	e2 := util.OneWaySyncDir(ThumbsFolder, bkThumbsFolder)
	return util.WrapErrors(e1, e2)
}

func syncExec(bkProjRoot string) error {
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

	// 删除文档
	if err := deleteInBKFiles(bk, bkBucketsDir, bkTemp); err != nil {
		return nil, err
	}

	// 必须先执行 deleteInBKFiles, 再执行 updateBKFiles
	if err := updateBKFiles(bk, bkBucketsDir, bkTemp); err != nil {
		return nil, err
	}

	// 必须先执行 updateBKFiles, 再执行 insertBKFiles
	if err := insertBKFiles(bk, bkBucketsDir); err != nil {
		return nil, err
	}
	return bkProjStat, nil
}

func deleteInBKFiles(bk *database.DB, bkBucketsDir, bkTemp string) error {
	rows, err := bk.Query(stmt.GetAllFiles)
	if err != nil {
		return err
	}
	for rows.Next() {
		bkFile, err := database.ScanFile(rows)
		if err != nil {
			return err
		}
		// 如果一个文档存在于备份仓库中，但不存在于主仓库中，
		// 那么说明该文档已被彻底删除，因此在备份仓库中也需要删除它。
		_, err = db.GetFileByID(bkFile.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
			if err2 := bk.DeleteFile(
				bkBucketsDir, bkTemp, thumbFilePath(bkFile.ID), &bkFile,
			); err2 != nil {
				return err2
			}
		}
		if err != nil {
			return err
		}
	}
	return util.WrapErrors(rows.Err(), rows.Close())
}

// 必须先执行 deleteInBKFiles, 删除多余的备份文档后,
// 这里的 dbFile 和 bkFile 就必然同时存在, 是同一个文档的新旧版本.
func updateBKFiles(bk *database.DB, bkBucketsDir, bkTemp string) error {
	rows, err := bk.Query(stmt.GetAllFiles)
	if err != nil {
		return err
	}
	for rows.Next() {
		bkFile, e1 := database.ScanFile(rows)
		dbFile, e2 := db.GetFileByID(bkFile.ID)
		if err := util.WrapErrors(e1, e2); err != nil {
			return err
		}

		// 对比除 Checksum 和 BucketName 以外的属性, 如果不一致则更新属性。
		if !filesHaveSameProperties(&bkFile, &dbFile) {
			if err := updateBKFile(&bkFile, &dbFile, bk, bkBucketsDir); err != nil {
				return err
			}
			bkFile.Name = dbFile.Name
		}

		// 对比 BucketName, 不一致则移动文档.
		if bkFile.BucketName != dbFile.BucketName {
			if err := moveBKFileToBucket(bkBucketsDir, dbFile.BucketName, &bkFile, bk); err != nil {
				return err
			}
			bkFile.BucketName = dbFile.BucketName
		}

		// 对比 checksum，不一致则拷贝覆盖。
		if bkFile.Checksum != dbFile.Checksum {
			if err := overwriteBKFile(bkBucketsDir, bkTemp, &dbFile, &bkFile, bk); err != nil {
				return err
			}
		}
	}
	return util.WrapErrors(rows.Err(), rows.Close())
}

func updateBKFile(bkFile, dbFile *File, bk *database.DB, bkBucketsDir string) error {
	// 如果文档名不一致, 还要重命名
	moved := new(MovedFile)
	if bkFile.Name != dbFile.Name {
		moved.Src = filepath.Join(bkBucketsDir, bkFile.BucketName, bkFile.Name)
		moved.Dst = filepath.Join(bkBucketsDir, bkFile.BucketName, dbFile.Name)
		if err := moved.Move(); err != nil {
			return err
		}
	}
	if err := bk.UpdateBackupFileInfo(dbFile); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

func insertBKFiles(bk *database.DB, bkBucketsDir string) error {
	rows, err := db.Query(stmt.GetAllFiles)
	if err != nil {
		return err
	}
	for rows.Next() {
		dbFile, err := database.ScanFile(rows)
		if err != nil {
			return err
		}
		if err := insertBKFile(bkBucketsDir, &dbFile, bk); err != nil {
			return err
		}
	}
	return util.WrapErrors(rows.Err(), rows.Close())
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
func syncBuckets(bkBucketsDir string, bk *database.DB) error {
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
			if err := renameBucket(oldName, newName, bkBucketsDir, bucket.ID); err != nil {
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

func renameBucket(oldName, newName, buckets string, bucketid int64) error {
	oldpath := filepath.Join(buckets, oldName)
	newpath := filepath.Join(buckets, newName)
	if err := os.Rename(oldpath, newpath); err != nil {
		return err
	}
	if err := db.UpdateBucketName(newName, bucketid); err != nil {
		err2 := os.Rename(newpath, oldpath)
		return util.WrapErrors(err, err2)
	}
	return nil
}

func moveBKFileToBucket(
	bkBucketsDir, newBucketName string, bkFile *File, bk *database.DB,
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

// 備份專案中的檔案 與 源專案中的檔案, 兩者的屬性相同嗎?
// 只對比一部分屬性. 在執行本函數之前, 應先同步 Checksum 和 BucketName,
// 因此在這裡不對比 Checksum, Size 和 BucketName.
// 另外, Checked 和 Damaged 也不對比.
func filesHaveSameProperties(bkFile, file *File) bool {
	return file.Name == bkFile.Name &&
		file.Notes == bkFile.Notes &&
		file.Keywords == bkFile.Keywords &&
		file.Like == bkFile.Like &&
		file.CTime == bkFile.CTime &&
		file.UTime == bkFile.UTime
}

func insertBKFile(bkBuckets string, file *File, bk *database.DB) error {
	_, err := bk.GetFileByID(file.ID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		dstFile := filepath.Join(bkBuckets, file.BucketName, file.Name)
		srcFile := filepath.Join(BucketsFolder, file.BucketName, file.Name)
		if err := util.CopyAndLockFile(dstFile, srcFile); err != nil {
			return err
		}
		if err := bk.InsertFileWithID(file); err != nil {
			err2 := os.Remove(dstFile)
			return util.WrapErrors(err, err2)
		}
	}
	return err
}

func overwriteBKFile(bkBucketsDir, bkTemp string, file, bkFile *File, bk *database.DB) error {
	// tempFile 把旧文档临时移动到安全的地方
	tempFile := MovedFile{
		Src: filepath.Join(bkBucketsDir, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkTemp, bkFile.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}

	// 复制新文档到备份仓库, 如果出错, 必须把旧文档移回原位.
	newFileDst := filepath.Join(bkBucketsDir, file.BucketName, file.Name)
	newFileSrc := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	if err := util.CopyAndLockFile(newFileDst, newFileSrc); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}

	// 更新数据库信息, 如果出错, 要删除 newFile 并把 tempFile 都移回原位.
	if err := bk.UpdateBackupFileInfo(file); err != nil {
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

// TODO: delete the thumb
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
	return db.DeleteFile(
		BucketsFolder, TempFolder, thumbFilePath(file.ID), &file.File)
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
