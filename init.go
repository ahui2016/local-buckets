package main

import (
	"os"
	"path/filepath"

	"github.com/ahui2016/local-buckets/database"
	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/util"
	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

type (
	Project       = model.Project
	Bucket        = model.Bucket
	File          = model.File
	FilePlus      = model.FilePlus
	MovedFile     = model.MovedFile
	ProjectInfo   = model.ProjectInfo
	ProjectStatus = model.ProjectStatus
	TX            = database.TX
)

const (
	ProjectTOML       = "project.toml"
	DatabaseFileName  = "project.db"
	WaitingFolderName = "waiting"
	BucketsFolderName = "buckets"
	TempFolderName    = "temp"
	PublicFolderName  = "public"
	ThumbsFolderName  = "thumbs"
	DotJPEG           = ".jpeg"
)

var (
	db                = new(database.DB)
	ProjectRoot       = filepath.Dir(util.GetExePath())
	ProjectConfig     *Project
	ProjectConfigPath = filepath.Join(ProjectRoot, ProjectTOML)
	DatabasePath      = filepath.Join(ProjectRoot, DatabaseFileName)
	WaitingFolder     = filepath.Join(ProjectRoot, WaitingFolderName)
	BucketsFolder     = filepath.Join(ProjectRoot, BucketsFolderName)
	TempFolder        = filepath.Join(ProjectRoot, TempFolderName)
	PublicFolder      = filepath.Join(ProjectRoot, PublicFolderName)
	ThumbsFolder      = filepath.Join(PublicFolder, ThumbsFolderName)
)

func init() {
	initProjectConfig()
	initDB()
	createFolders()
}

func initDB() {
	lo.Must0(db.Open(DatabasePath, ProjectConfig))
}

func createFolders() {
	folders := []string{
		BucketsFolder,
		WaitingFolder,
		TempFolder,
		PublicFolder,
		ThumbsFolder,
	}
	for _, folder := range folders {
		lo.Must0(util.MkdirIfNotExists(folder))
	}
}

func createBucketFolder(bucketID string) error {
	bucketPath := filepath.Join(BucketsFolder, bucketID)
	return util.MkdirIfNotExists(bucketPath)
}

func readProjectConfig() {
	data := lo.Must(os.ReadFile(ProjectConfigPath))
	lo.Must0(toml.Unmarshal(data, &ProjectConfig))
}

func writeProjectConfig() error {
	return util.WriteTOML(ProjectConfig, ProjectConfigPath)
}

func initProjectConfig() {
	if util.PathIsNotExist(ProjectConfigPath) {
		title := filepath.Base(ProjectRoot)
		cipherkey := database.DefaultCipherKey()
		ProjectConfig = model.NewProject(title, cipherkey)
		lo.Must0(writeProjectConfig())
		return
	}
	readProjectConfig()
}

// 更新备份时间
func projCfgBackupNow(bkProjStat *ProjectStatus) error {
	ProjectConfig.LastBackupAt = model.Now()
	err1 := writeProjectConfig()
	bkProjCfgPath := filepath.Join(bkProjStat.Root, ProjectTOML)
	bkProjStat.LastBackupAt = ProjectConfig.LastBackupAt
	err2 := util.WriteTOML(bkProjStat.Project, bkProjCfgPath)
	return util.WrapErrors(err1, err2)
}

func addBKProjToConfig(bkProjRoot string) error {
	ProjectConfig.BackupProjects = append(ProjectConfig.BackupProjects, bkProjRoot)
	return util.WriteTOML(ProjectConfig, ProjectConfigPath)
}

func deleteBKProjFromConfig(bkProj string) error {
	ProjectConfig.BackupProjects = lo.Reject(
		ProjectConfig.BackupProjects, func(x string, _ int) bool {
			return x == bkProj
		})
	return util.WriteTOML(ProjectConfig, ProjectConfigPath)
}

func thumbFilePath(filename string) string {
	basename := filepath.Base(filename)
	return filepath.Join(ThumbsFolder, basename+DotJPEG)
}
