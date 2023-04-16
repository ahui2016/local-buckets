package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
	ProjectStatus = model.ProjectStatus
	BucketStatus  = model.BucketStatus
	TX            = database.TX
)

const (
	GB  = model.GB
	Day = model.Day
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
	DotTOML           = ".toml"
)

var (
	db                *database.DB
	ProjectConfig     *Project
	ProjectRoot       = filepath.Dir(util.GetExePath())
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
	fmt.Println(ProjectConfig)
	initDB()
	createFolders()
}

func initDB() {
	db = lo.Must1(database.OpenDB(DatabasePath, ProjectConfig))
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

func readProjCfgFrom(cfgPath string) (cfg Project, err error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return
	}
	err = toml.Unmarshal(data, &cfg)
	return
}

func writeProjectConfig() error {
	return util.WriteTOML(ProjectConfig, ProjectConfigPath)
}

func initProjectConfig() {
	if util.PathNotExists(ProjectConfigPath) {
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

func thumbFilePath(fileID int64) string {
	filename := strconv.FormatInt(fileID, 10)
	return filepath.Join(ThumbsFolder, filename)
}

func tempFilePath(fileID int64) string {
	filename := strconv.FormatInt(fileID, 10)
	return filepath.Join(TempFolder, filename)
}
