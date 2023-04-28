package database

import (
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/stmt"
	"github.com/ahui2016/local-buckets/util"
	"github.com/samber/lo"
	_ "modernc.org/sqlite"
)

type (
	Base64String     = string
	Bucket           = model.Bucket
	File             = model.File
	FilePlus         = model.FilePlus
	FileExportImport = model.FileExportImport
	Project          = model.Project
	ProjectStatus    = model.ProjectStatus
	BucketStatus     = model.BucketStatus
	MovedFile        = model.MovedFile
	ErrSameNameFiles = model.ErrSameNameFiles
)

type DB struct {
	DB               *sql.DB
	Path             string // 数据库的路径
	IsBackup         bool
	RecentFilesLimit int64
	cipherKey        HexString
	aesgcm           cipher.AEAD
}

func OpenDB(dbPath string, projCfg *Project) (*DB, error) {
	const turnOnForeignKeys = "?_pragma=foreign_keys(1)"
	sqlDB, err := sql.Open("sqlite", dbPath+turnOnForeignKeys)
	if err != nil {
		return nil, err
	}
	db := &DB{
		DB:               sqlDB,
		Path:             dbPath,
		IsBackup:         projCfg.IsBackup,
		RecentFilesLimit: projCfg.RecentFilesLimit,
		cipherKey:        projCfg.CipherKey,
		aesgcm:           nil,
	}
	err = db.Exec(stmt.CreateTables)
	return db, err
}

func (db *DB) Exec(query string, args ...any) (err error) {
	_, err = db.DB.Exec(query, args...)
	return
}

func (db *DB) QueryRow(query string, args ...any) *sql.Row {
	return db.DB.QueryRow(query, args...)
}

func (db *DB) Query(query string, args ...any) (*sql.Rows, error) {
	return db.DB.Query(query, args...)
}

func (db *DB) MustBegin() *sql.Tx {
	return lo.Must(db.DB.Begin())
}

// GetInt1 gets one Integer value from the database.
func (db *DB) GetInt1(query string, arg ...any) (int64, error) {
	return getInt1(db.DB, query, arg...)
}

func (db *DB) IsLoggedIn() bool {
	return db.aesgcm != nil
}

func (db *DB) Logout() {
	db.aesgcm = nil
}

func (db *DB) SetAESGCM(password string) (realKey []byte, err error) {
	aesgcm := newGCM(password)
	cipherBytes, err := hex.DecodeString(db.cipherKey)
	if err != nil {
		return nil, err
	}
	realKey, err = decrypt(cipherBytes, aesgcm)
	if err == nil {
		db.aesgcm = aesgcm
	}
	return
}

func (db *DB) ChangePassword(oldPwd, newPwd string) (HexString, error) {
	realKey, err := db.SetAESGCM(oldPwd)
	if err != nil {
		return "", err
	}
	aesgcm := newGCM(newPwd)
	encryptedKey := lo.Must(encrypt(realKey[:], aesgcm))
	db.aesgcm = aesgcm
	db.cipherKey = hex.EncodeToString(encryptedKey)
	return db.cipherKey, nil
}

// autoGetBuckets 根据 db.IsLoggedIn 自动获取公开仓库或全部仓库
func (db *DB) autoGetBuckets() ([]*Bucket, error) {
	query := lo.Ternary(db.IsLoggedIn(), stmt.GetAllBuckets, stmt.GetPublicBuckets)
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}

func (db *DB) GetAllBuckets() ([]*Bucket, error) {
	rows, err := db.Query(stmt.GetAllBuckets)
	if err != nil {
		return nil, err
	}
	return scanBuckets(rows)
}

func (db *DB) GetBucket(id int64) (Bucket, error) {
	row := db.QueryRow(stmt.GetBucket, id)
	return scanBucket(row)
}

func (db *DB) GetBucketByName(name string) (Bucket, error) {
	row := db.QueryRow(stmt.GetBucketByName, name)
	return scanBucket(row)
}

func (db *DB) BucketExists(name string) (bool, error) {
	_, err := db.GetBucketByName(name)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return true, err // 如果出错, 第一个值不可用, 如果无错, 第一个值必然为真.
}

func (db *DB) InsertBucket(form *model.CreateBucketForm) (*Bucket, error) {
	bucket, err := model.NewBucket(form)
	if err != nil {
		return nil, err
	}
	if err = insertBucket(db.DB, bucket); err != nil {
		return nil, err
	}
	b, err := db.GetBucketByName(bucket.Name)
	return &b, err
}

func (db *DB) InsertBucketWithID(bucket *Bucket) error {
	return insertBucketWithID(db.DB, bucket)
}

func (db *DB) InsertFile(file *File) error {
	return insertFile(db.DB, file)
}

// InsertAndReturnFile 主要用于同名檔案冲突时的逐一处理.
func (db *DB) InsertAndReturnFile(file *File) (*File, error) {
	if err := insertFile(db.DB, file); err != nil {
		return nil, err
	}
	f, err := db.GetFileByChecksum(file.Checksum)
	return &f, err
}

func (db *DB) InsertFileWithID(file *File) error {
	return insertFileWithID(db.DB, file)
}

// 该函数可能可以删除。
// 注意, 在使用该函数之前, 请先使用 db.CheckSameFiles() 检查全部等待处理的檔案.
func (db *DB) insertFiles(files []*File) error {
	tx := db.MustBegin()
	defer tx.Rollback()

	for _, file := range files {
		if err := insertFile(db.DB, file); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) UpdateFileContent(file *File) error {
	return db.Exec(
		stmt.UpdateFileContent, file.Checksum, file.Size, file.UTime, file.ID)
}

func (db *DB) UpdateFileInfo(file *File) error {
	return db.Exec(stmt.UpdateFileInfo, file.Name, file.Notes,
		file.Keywords, file.Type, file.Like, file.CTime, file.UTime, file.ID)
}

func (db *DB) MoveFileToBucket(fileID int64, bucketName string) error {
	return db.Exec(stmt.MoveFileToBucket, bucketName, fileID)
}

// 主要用于仓库之间移动文档, 因加密解密而使 checksum 发生变化.
func (db *DB) UpdateChecksumAndBucket(fileID int64, checksum, bucketName string) error {
	return db.Exec(stmt.UpdateChecksumAndBucket, checksum, bucketName, fileID)
}

// CheckSameFiles 检查有无同名/相同内容的檔案,
// 发现相同内容的檔案时, 记录全部重复檔案后再汇总返回错误,
// 但发现同名檔案时, 则立即返回错误 (因为前端需要对同名檔案进行逐个处理).
func (db *DB) CheckSameFiles(files []*File) (allErr error) {
	for _, file := range files {
		if err := db.checkSameFile(file); err != nil {
			if e, ok := err.(ErrSameNameFiles); ok {
				return e
			}
			allErr = util.WrapErrors(allErr, err)
		}
	}
	return
}

// CheckSameFile 发现有相同檔案 (同名或同内容) 时返回错误,
// 未发现相同檔案则返回 nil 或其他错误.
func (db *DB) checkSameFile(file *File) error {
	if err := db.CheckSameChecksum(file); err != nil {
		return err
	}
	return db.CheckSameFilename(file.Name)
}

// 有同名檔案时返回 ErrSameNameFiles, 无同名檔案则返回 nil 或其他错误.
func (db *DB) CheckSameFilename(name string) error {
	same, err := db.GetFileByName(name)
	if err == nil && len(same.Name) > 0 {
		return model.NewErrSameNameFiles(same)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return err
}

// 有相同内容的檔案时返回 error(相同内容的檔案已存在),
// 无相同内容的檔案则返回 nil 或其他错误.
func (db *DB) CheckSameChecksum(file *File) error {
	same, err := db.GetFileByChecksum(file.Checksum)
	if err == nil && len(same.Name) > 0 {
		return fmt.Errorf(
			"相同内容的檔案已存在: %s ↔ %s/%s", file.Name, same.BucketName, same.Name)
	}
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	return err
}

func (db *DB) GetFileByChecksum(checksum string) (File, error) {
	row := db.QueryRow(stmt.GetFileByChecksum, checksum)
	return ScanFile(row)
}

func (db *DB) getFilesByChecksum(checksum string) (files []*File, err error) {
	rows, err := db.Query(stmt.GetFileByChecksum, checksum)
	if err != nil {
		return
	}
	return scanFiles(rows)
}

func (db *DB) GetFileByName(name string) (File, error) {
	row := db.QueryRow(stmt.GetFileByName, name)
	return ScanFile(row)
}

func (db *DB) GetFileByID(id int64) (File, error) {
	row := db.QueryRow(stmt.GetFileByID, id)
	return ScanFile(row)
}

func (db *DB) GetFilePlus(id int64) (file FilePlus, err error) {
	row := db.QueryRow(stmt.GetFilePlus, id)
	if file, err = scanFilePlus(row); err != nil {
		return
	}
	return
}

func (db *DB) GetFilePlusByName(name string) (file FilePlus, err error) {
	row := db.QueryRow(stmt.GetFilePlusByName, name)
	if file, err = scanFilePlus(row); err != nil {
		return
	}
	file.Checksum = ""
	return
}

// TODO: 要刪除, 使用該函數的地方改為 iterator
func (db *DB) GetAllFiles() (files []*File, err error) {
	return getFiles(db.DB, stmt.GetAllFiles)
}

func (db *DB) GetRecentFiles() (files []*FilePlus, err error) {
	query := lo.Ternary(db.IsLoggedIn(), stmt.GetAllRecentFiles, stmt.GetPublicRecentFiles)
	if files, err = getFilesPlus(db.DB, query, db.RecentFilesLimit); err != nil {
		return
	}
	files = RemoveChecksum(files)
	return
}

func (db *DB) RecentFilesInBucket(id int64) (files []*FilePlus, err error) {
	query := lo.Ternary(db.IsLoggedIn(), stmt.AllRecentFilesInBucket, stmt.PublicRecentFilesInBucket)
	if files, err = getFilesPlus(db.DB, query, id, db.RecentFilesLimit); err != nil {
		return
	}
	files = RemoveChecksum(files)
	return
}

func (db *DB) GetRecentPics() (files []*FilePlus, err error) {
	query := lo.Ternary(db.IsLoggedIn(), stmt.GetAllRecentPics, stmt.GetPublicRecentPics)
	if files, err = getFilesPlus(db.DB, query, db.RecentFilesLimit); err != nil {
		return
	}
	files = RemoveChecksum(files)
	return
}

func (db *DB) RecentPicsInBucket(id int64) (files []*FilePlus, err error) {
	query := lo.Ternary(db.IsLoggedIn(), stmt.AllRecentPicsInBucket, stmt.PublicRecentPicsInBucket)
	if files, err = getFilesPlus(db.DB, query, id, db.RecentFilesLimit); err != nil {
		return
	}
	files = RemoveChecksum(files)
	return
}

func RemoveChecksum(files []*FilePlus) []*FilePlus {
	for i := range files {
		files[i].Checksum = ""
	}
	return files
}

func (db *DB) GetProjStat(projCfg *Project) (ProjectStatus, error) {
	totalSize, e1 := getInt1(db.DB, stmt.TotalSize)
	allFilesCount, e2 := getInt1(db.DB, stmt.CountAllFiles)
	needCheckCount, e3 := countFilesNeedCheck(db.DB, projCfg.CheckInterval)
	damagedCount, e4 := getInt1(db.DB, stmt.CountDamagedFiles)
	err := util.WrapErrors(e1, e2, e3, e4)
	projStat := ProjectStatus{
		Project:           projCfg,
		Root:              filepath.Dir(db.Path),
		TotalSize:         totalSize,
		FilesCount:        allFilesCount,
		WaitingCheckCount: needCheckCount,
		DamagedCount:      damagedCount,
	}
	return projStat, err
}

// AllBucketsStatus 根据 db.IsLoggedIn 選擇获取公开仓库或全部仓库的狀態
func (db *DB) AllBucketsStatus() (statusList []BucketStatus, err error) {
	buckets, err := db.autoGetBuckets()
	if err != nil {
		return nil, err
	}
	for _, bucket := range buckets {
		totalSize, e1 := getInt1(db.DB, stmt.BucketTotalSize, bucket.ID)
		filesCount, e2 := getInt1(db.DB, stmt.BucketCountFiles, bucket.ID)
		if err := util.WrapErrors(e1, e2); err != nil {
			return nil, err
		}
		bucketStat := BucketStatus{
			Bucket:     bucket,
			TotalSize:  totalSize,
			FilesCount: filesCount,
		}
		statusList = append(statusList, bucketStat)
	}
	return
}

// UpdateBackupFileInfo 更新一个文档的大多数信息, 但不更新 Checked 和 Damaged.
func (db *DB) UpdateBackupFileInfo(file *File) error {
	return db.Exec(stmt.UpdateBackupFileInfo, file.Checksum, file.BucketName,
		file.Name, file.Notes, file.Keywords, file.Size, file.Type,
		file.Like, file.CTime, file.UTime, file.Deleted, file.ID)
}

// DeleteFile 刪除檔案, 包括從數據庫中刪除和從硬碟中刪除.
func (db *DB) DeleteFile(bucketsDir, tempDir, thumbPath string, file *File) error {
	tempFile := MovedFile{
		Src: filepath.Join(bucketsDir, file.BucketName, file.Name),
		Dst: filepath.Join(tempDir, file.Name),
	}
	if err := tempFile.Move(); err != nil {
		return err
	}
	if err := db.Exec(stmt.DeleteFile, file.ID); err != nil {
		err2 := tempFile.Rollback()
		return util.WrapErrors(err, err2)
	}
	_ = os.Remove(thumbPath)
	return os.Remove(tempFile.Dst)
}

func (db *DB) DeleteBucket(bucketid int64) error {
	return db.Exec(stmt.DeleteBucket, bucketid)
}

func (db *DB) UpdateBucketName(newName string, bucketid int64) error {
	return db.Exec(stmt.UpdateBucketName, newName, bucketid)
}

func (db *DB) UpdateBucketTitle(bucket *Bucket) error {
	return db.Exec(stmt.UpdateBucketTitle, bucket.Title, bucket.Subtitle, bucket.ID)
}

// EncryptFile 读取 srcPath 的文件, 加密后保存到 dstPath.
func (db *DB) EncryptFile(srcPath, dstPath string, perm fs.FileMode) error {
	if util.PathExists(dstPath) {
		return fmt.Errorf("file exists: %s", dstPath)
	}
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	encrypted, err := encrypt(data, db.aesgcm)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, encrypted, perm)
}

// DecryptSaveFile 读取 srcPath 的文件, 解密后保存到 dstPath.
func (db *DB) DecryptSaveFile(srcPath, dstPath string, perm fs.FileMode) error {
	content, err := db.DecryptFile(srcPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstPath, content, perm)
}

func (db *DB) DecryptFile(filePath string) ([]byte, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return decrypt(data, db.aesgcm)
}

func (db *DB) GetDamagedFiles() ([]*FilePlus, error) {
	return getFilesPlus(db.DB, stmt.GetDamagedFiles)
}

// GetFilesNeedCheck 获取需要检查的文件, checkInterval 的单位是秒.
func (db *DB) GetFilesNeedCheck(checkInterval int64) ([]*File, error) {
	// 如果一个文件的上次校验日期小于(早于) needCheckDate, 那么这个文件就需要再次校验。
	needCheckDateUnix := time.Now().Unix() - checkInterval
	needCheckDate := time.Unix(needCheckDateUnix, 0).Format(model.RFC3339)
	return getFiles(db.DB, stmt.GetFilesNeedCheck, needCheckDate)
}

func (db *DB) SetFileCheckedDamaged(file *File) error {
	return db.Exec(stmt.CheckFile, file.Checked, file.Damaged, file.ID)
}

func (db *DB) SearchFiles(pattern, fileType string, limit int64) (files []*FilePlus, err error) {
	query := stmt.SearchAllFiles
	if db.IsLoggedIn() && fileType == "" {
		query = stmt.SearchPublicFiles
	}
	if fileType == "image" {
		query = stmt.SearchAllPics
	}
	if fileType == "image" && db.IsLoggedIn() {
		query = stmt.SearchPublicPics
	}

	pattern = "%" + pattern + "%"
	return getFilesPlus(db.DB, query, pattern, pattern, pattern, limit)
}
