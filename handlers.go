package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ahui2016/local-buckets/database"
	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/ahui2016/local-buckets/util"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/pelletier/go-toml/v2"
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
	time.Sleep(time.Second)
	return c.Next()
}

func parseValidate(form any, c *fiber.Ctx) error {
	if err := c.BodyParser(form); err != nil {
		return err
	}
	return validate.Struct(form)
}

func getProjectInfo(c *fiber.Ctx) error {
	return c.JSON(ProjectInfo{Project: ProjectConfig, Path: ProjectRoot})
}

func getProjectStatus(c *fiber.Ctx) error {
	projStat, err := db.GetProjStat(ProjectConfig)
	if err != nil {
		return err
	}
	return c.JSON(projStat)
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
	return writeProjectConfig()
}

func adminLogin(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	_, err := db.SetAESGCM(form.Text)
	return err
}

// TODO: 输入密码后才包含加密仓库
func autoGetBuckets(c *fiber.Ctx) error {
	buckets, err := db.AutoGetBuckets()
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

	// 这个 fileInDB 主要是为了获取 File.ID 和 BucketName.
	fileInDB, err := db.GetFileByName(file.Name)
	if err != nil {
		return err
	}
	file.ID = fileInDB.ID
	file.BucketName = fileInDB.BucketName

	waitingFile.Dst = filepath.Join(BucketsFolder, file.BucketName, file.Name)

	// 以上是收集信息及检查错误
	// 以下开始操作文档和数据库

	// tempFile 把旧文档临时移动到安全的地方
	// 在文档名区分大小写的系统里, 要注意 file.Name 与 fileInDB.Name 可能不同.
	tempFile := MovedFile{
		Src: filepath.Join(BucketsFolder, file.BucketName, fileInDB.Name),
		Dst: filepath.Join(TempFolder, fileInDB.Name),
	}
	if err = tempFile.Move(); err != nil {
		return err
	}

	// 移动新文档进仓库, 如果出错, 必须把旧文档移回原位.
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
	bucket, err2 := db.GetBucketByName(form.Text) // bucketName = form.Text
	if err := util.WrapErrors(err1, err2); err != nil {
		return err
	}
	files, err := checkAndGetWaitingFiles()
	if err != nil {
		return err
	}

	// 以上是检查阶段
	// 以下是实际执行阶段

	files = setBucketName(bucket.Name, files)
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
		Dst: filepath.Join(BucketsFolder, file.BucketName, file.Name),
	}
	err := movedFile.Move()
	return movedFile, err
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

// TODO: 在加密仓库与公开仓库之间移动文档
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
		Src: filepath.Join(BucketsFolder, file.BucketName, file.Name),
		Dst: filepath.Join(BucketsFolder, form.BucketName, file.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if err := db.MoveFileToBucket(form.FileID, form.BucketName); err != nil {
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
	return nil
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
	if util.PathIsNotExist(bkProjRoot) {
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
	bkProjCfgPath := filepath.Join(bkProjRoot, ProjectTOML)
	if err := util.WriteTOML(bkProjCfg, bkProjCfgPath); err != nil {
		return err
	}
	exePath := util.GetExePath()
	exeName := filepath.Base(exePath)
	bkProjExePath := filepath.Join(bkProjRoot, exeName)
	if err := util.CopyFile(bkProjExePath, exePath); err != nil {
		return err
	}
	bkProjBucketsDir := filepath.Join(bkProjRoot, BucketsFolderName)
	return util.Mkdir(bkProjBucketsDir)
}

func getBKProjStat(c *fiber.Ctx) error {
	form := new(model.OneTextForm)
	if err := parseValidate(form, c); err != nil {
		return err
	}
	bkProjRoot := form.Text
	if util.PathIsNotExist(bkProjRoot) {
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
func openBackupDB(bkProjRoot string) (bk *database.DB, bkProjStat *ProjectStatus, err error) {
	bkPath := filepath.Join(bkProjRoot, DatabaseFileName)
	bkProjCfgPath := filepath.Join(bkProjRoot, ProjectTOML)
	data, err := os.ReadFile(bkProjCfgPath)
	if err != nil {
		return
	}
	bk = new(database.DB)
	bkProjStat = new(ProjectStatus)
	var bkProjCfg Project
	if err = toml.Unmarshal(data, &bkProjCfg); err != nil {
		return
	}
	if err = bk.Open(bkPath, &bkProjCfg); err != nil {
		return
	}
	*bkProjStat, err = bk.GetProjStat(&bkProjCfg)
	return
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
	return projCfgBackupNow(bkProjStat)
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

	if err := checkBackupDiskUsage(bkProjRoot, bkProjStat, &projStat); err != nil {
		return nil, err
	}

	bkBucketsDir := filepath.Join(bkProjRoot, BucketsFolderName)
	bkTemp := filepath.Join(bkProjRoot, TempFolderName)

	// 先处理仓库, 包括新建或删除仓库资料夹.
	if err := syncBuckets(bkBucketsDir, bk); err != nil {
		return nil, err
	}

	dbFiles, e1 := db.GetAllFiles()
	bkFiles, e2 := bk.GetAllFiles()
	if err := util.WrapErrors(e1, e2); err != nil {
		return nil, err
	}

	for _, bkFile := range bkFiles {
		// 如果一个文档存在于备份仓库中，但不存在于主仓库中，
		// 那么说明该文档已被彻底删除，因此在备份仓库中也需要删除它。
		dbFile, err := db.GetFileByID(bkFile.ID)
		if errors.Is(err, sql.ErrNoRows) {
			err = nil
			if err2 := bk.DeleteFile(bkBucketsDir, bkTemp, bkFile); err2 != nil {
				return nil, err2
			}
			continue // 这句是必须的
		}
		if err != nil {
			return nil, err
		}
		// 从这里开始 dbFile 和 bkFile 同时存在, 是同一个文档的新旧版本.

		// 对比 BucketName, 不一致则移动文档.
		// Bug: 有很低的可能性发生文档名称冲突，知道就行，暂时可以偷懒不处理。
		if dbFile.BucketName != bkFile.BucketName {
			if err := moveBKFileToBucket(
				bkBucketsDir, dbFile.BucketName, bkFile, bk); err != nil {
				return nil, err
			}
			// 这里不能 continue
		}

		// 对比 checksum，不一致则拷贝覆盖。
		// Bug: 有很低的可能性发生 checksum 冲突，知道就行，暂时可以偷懒不处理。
		if dbFile.Checksum != bkFile.Checksum {
			if err := overwriteBKFile(bkBucketsDir, bkTemp, &dbFile, bkFile, bk); err != nil {
				return nil, err
			}
			continue // 这句是必须的, 因为在 overwriteBKFile 里会更新文档的全部属性.
		}

		// 前面已经对比过 Checksum 和 BucketName, 现在对比其他属性，
		// 如果不一致则更新属性。
		if filesHaveSameProperties(bkFile, &dbFile) {
			continue
		}
		if err := bk.UpdateBackupFileInfo(&dbFile); err != nil {
			return nil, err
		}
	}

	// 如果一个文档存在于主仓库中，但不存在于备份仓库中，则直接拷贝。
	// 这一步应该在更新 checksum 不同的文档之后，避免 checksum 冲突。
	for _, file := range dbFiles {
		if err := insertBKFile(bkBucketsDir, file, bk); err != nil {
			return nil, err
		}
	}

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
	if err := bk.Exec(stmt.MoveFileToBucket, newBucketName, bkFile.ID); err != nil {
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
		file.Type == bkFile.Type &&
		file.Like == bkFile.Like &&
		file.CTime == bkFile.CTime &&
		file.UTime == bkFile.UTime &&
		file.Deleted == bkFile.Deleted
}

func insertBKFile(bkBuckets string, file *File, bk *database.DB) error {
	_, err := bk.GetFileByID(file.ID)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
		dstFile := filepath.Join(bkBuckets, file.BucketName, file.Name)
		srcFile := filepath.Join(BucketsFolder, file.BucketName, file.Name)
		if err := util.CopyFile(dstFile, srcFile); err != nil {
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
	if err := util.CopyFile(newFileDst, newFileSrc); err != nil {
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
