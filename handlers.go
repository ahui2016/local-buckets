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

	// 这个 fileInDB 主要是为了获取 File.ID 和 BucketID, BucketName.
	fileInDB, err := db.GetFileByName(file.Name)
	if err != nil {
		return err
	}
	file.ID = fileInDB.ID
	file.BucketID = fileInDB.BucketID
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

	files = setBucketID(bucket.ID, bucket.Name, files)
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

func setBucketID(bucketID int64, bucketName string, files []*File) []*File {
	for _, file := range files {
		file.BucketID = bucketID
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
	file, err1 := db.GetFileByID(form.FileID)
	bucket, err2 := db.GetBucket(form.BucketID)
	if err := util.WrapErrors(err1, err2); err != nil {
		return err
	}
	moved := MovedFile{
		Src: filepath.Join(BucketsFolder, file.BucketName, file.Name),
		Dst: filepath.Join(BucketsFolder, bucket.Name, file.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if err := db.MoveFileToBucket(form.FileID, bucket.ID, bucket.Name); err != nil {
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

// Bug: 不能使用 tx
// 同步仓库信息到备份仓库 (包括仓库资料夹重命名)
func syncBuckets(bk *database.DB, bkTX TX) error {
	dbBuckets, e1 := db.GetAllBuckets()
	bkBuckets, e2 := bk.GetAllBuckets()
	if err := util.WrapErrors(e1, e2); err != nil {
		return err
	}

	for _, bucket := range dbBuckets {
		bkBucket, err := bk.GetBucket(bucket.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}
		// 如果仓库资料夹名称发生了改变, 则备份仓库的资料夹也要重命名
		oldName := bkBucket.Name
		newName := bucket.Name
		if bkBucket.Name != bucket.Name {
			if err := os.Rename(oldName, newName); err != nil {
				return err
			}
		}
		if bkBucket.Name != bucket.Name ||
			bkBucket.Title != bucket.Title ||
			bkBucket.Subtitle != bucket.Subtitle {
			if err := database.UpdateBucketInfo(bkTX, bucket); err != nil {
				err2 := os.Rename(newName, oldName)
				return util.WrapErrors(err, err2)
			}
		}
	}

}

// Bug: 不能使用 tx
// syncToBackupProject 以源仓库为准单向同步，
// 最终效果相当于清空备份仓库后把主仓库的全部文档复制到备份仓库。
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

	dbFiles, e1 := db.GetAllFiles()
	bkFiles, e2 := bk.GetAllFiles()
	if err := util.WrapErrors(e1, e2); err != nil {
		return nil, err
	}

	bkTX := bk.MustBegin()
	defer bkTX.Rollback()

	bkBuckets := filepath.Join(bkProjRoot, BucketsFolderName)
	bkTemp := filepath.Join(bkProjRoot, TempFolderName)

	// 如果一个文档存在于备份仓库中，但不存在于主仓库中，
	// 那么说明该文档已被彻底删除，因此在备份仓库中也需要删除它。
	for _, bkFile := range bkFiles {
		if err := deleteBKFile(bkBuckets, bkTemp, bkFile, bkTX); err != nil {
			return nil, err
		}
	}

	// 如果一个文档同时存在于两个专案中, 则对比其 bucketid, 不一致则移动文档.
	// Bug: 有很低的可能性发生文档名称冲突，知道就行，暂时可以偷懒不处理。
	for _, file := range dbFiles {
		bkFile, err := bk.GetFileByID(file.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if file.BucketID != bkFile.BucketID {
			if err := moveBKFileToBucket(bkBuckets, file.BucketName, &bkFile, bkTX); err != nil {
				return nil, err
			}
		}
	}

	// 如果一个文档存在于两个专案中，则进一步对比其 checksum，不一致则拷贝覆盖。
	// Bug: 有很低的可能性发生 checksum 冲突，知道就行，暂时可以偷懒不处理。
	for _, file := range dbFiles {
		bkFile, err := bk.GetFileByID(file.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if err := overwriteBKFile(bkBuckets, bkTemp, file, &bkFile, bkTX); err != nil {
			return nil, err
		}
	}

	// 前面已经对比过 checksum 和 bucketid, 现在对比其他属性，
	// 如果不一致则更新属性。
	for _, file := range dbFiles {
		bkFile, err := bk.GetFileByID(file.ID)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if filesHaveSameProperties(&bkFile, file) {
			continue
		}
		if err := database.UpdateBackupFileInfo(bkTX, file); err != nil {
			return nil, err
		}
	}

	// 如果一个文档存在于主仓库中，但不存在于备份仓库中，则直接拷贝。
	// 这一步应该在更新 checksum 不同的文档之后，避免 checksum 冲突。
	for _, file := range dbFiles {
		if err := insertBKFile(bkBuckets, file, bkTX); err != nil {
			return nil, err
		}
	}

	err = bkTX.Commit()
	return bkProjStat, err
}

func moveBKFileToBucket(bkBuckets, newBucketName string, bkFile *File, bkTX TX) error {
	moved := MovedFile{
		Src: filepath.Join(bkBuckets, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkBuckets, newBucketName, bkFile.Name),
	}
	if err := moved.Move(); err != nil {
		return err
	}
	if _, err := bkTX.Exec(stmt.MoveFileToBucket, newBucketName, bkFile.ID); err != nil {
		err2 := moved.Rollback()
		return util.WrapErrors(err, err2)
	}
	return nil
}

// 備份專案中的檔案 與 源專案中的檔案, 兩者的屬性相同嗎?
// 只對比一部分屬性. 在執行本函數之前, 應先同步 checksum 和 bucketid,
// 因此在這裡不對比 checksum, size 和 bucketid.
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

func deleteBKFile(bkBuckets, bkTemp string, bkFile *File, bkTX TX) error {
	_, err := db.GetFileByID(bkFile.ID)
	if errors.Is(err, sql.ErrNoRows) {
		if err2 := database.DeleteFile(bkTX, bkBuckets, bkTemp, bkFile); err2 != nil {
			return err2
		}
	}
	return err
}

func insertBKFile(bkBuckets string, file *File, bkTX TX) error {
	_, err := database.TxGetFileByID(bkTX, file.ID)
	if errors.Is(err, sql.ErrNoRows) {
		dstFile := filepath.Join(bkBuckets, file.Name)
		srcFile := filepath.Join(BucketsFolder, file.Name)
		if err := util.CopyFile(dstFile, srcFile); err != nil {
			return err
		}
		if err := database.InsertFileWithID(bkTX, file); err != nil {
			err2 := os.Remove(dstFile)
			return util.WrapErrors(err, err2)
		}
	}
	return err
}

func overwriteBKFile(bkBuckets, bkTemp string, file, bkFile *File, bkTX TX) error {
	// tempFile 把旧文档临时移动到安全的地方
	tempFile := MovedFile{
		Src: filepath.Join(bkBuckets, bkFile.BucketName, bkFile.Name),
		Dst: filepath.Join(bkTemp, bkFile.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}

	// 复制新文档到备份仓库, 如果出错, 必须把旧文档移回原位.
	newFileDst := filepath.Join(bkBuckets, file.BucketName, file.Name)
	newFileSrc := filepath.Join(BucketsFolder, file.BucketName, file.Name)
	if err := util.CopyFile(newFileDst, newFileSrc); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}

	// 更新数据库信息, 如果出错, 要删除 newFile 并把 tempFile 都移回原位.
	if err := database.UpdateBackupFileInfo(bkTX, file); err != nil {
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
