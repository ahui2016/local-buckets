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
	Project   = model.Project
	Bucket    = model.Bucket
	File      = model.File
	MovedFile = model.MovedFile
)

const (
	ProjectTOML       = "project.toml"
	DatabaseFileName  = "project.db"
	WaitingFolderName = "waiting"
	BucketsFolderName = "buckets"
	TempFolderName    = "temp" // TODO: 删除
	TempFilesJsonName = "temp_files.json"
	PublicFolderName  = "public"
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
	TempFilesJsonPath = filepath.Join(TempFolder, TempFilesJsonName)
	PublicFolder      = filepath.Join(ProjectRoot, PublicFolderName)
)

func init() {
	initProjectConfig()
	initDB()
	createFolders()
}

func initDB() {
	lo.Must0(db.Open(DatabasePath, ProjectRoot, ProjectConfig.CipherKey))
}

func createFolders() {
	util.MkdirIfNotExists(WaitingFolder, 0)
	util.MkdirIfNotExists(BucketsFolder, 0)
	util.MkdirIfNotExists(TempFolder, 0)
	util.MkdirIfNotExists(PublicFolder, 0)
}

func createBucketFolder(bucketID string) {
	path := filepath.Join(BucketsFolder, bucketID)
	util.MkdirIfNotExists(path, util.ReadonlyFolderPerm)
}

func readProjectConfig() {
	data := lo.Must(os.ReadFile(ProjectConfigPath))
	lo.Must0(toml.Unmarshal(data, &ProjectConfig))
}

func writeProjectConfig() {
	util.WriteTOML(ProjectConfig, ProjectConfigPath)
}

func initProjectConfig() {
	if util.PathIsNotExist(ProjectConfigPath) {
		title := filepath.Base(ProjectRoot)
		cipherkey := database.DefaultCipherKey()
		ProjectConfig = model.NewProject(title, cipherkey)
		writeProjectConfig()
		return
	}
	readProjectConfig()
}
