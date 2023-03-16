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

const (
	ProjectTOML       = "project.toml"
	DatabaseFileName  = "project.db"
	WaitingFolderName = "waiting"
	BucketsFolderName = "buckets"
	PublicFolderName  = "public"
)

var (
	db                = new(database.DB)
	ProjectPath       = filepath.Dir(util.GetExePath())
	ProjectConfig     *model.Project
	ProjectConfigPath = filepath.Join(ProjectPath, ProjectTOML)
	DatabasePath      = filepath.Join(ProjectPath, DatabaseFileName)
	WaitingFolder     = filepath.Join(ProjectPath, WaitingFolderName)
	BucketsFolder     = filepath.Join(ProjectPath, BucketsFolderName)
	PublicFolder      = filepath.Join(ProjectPath, PublicFolderName)
)

func init() {
	initProjectConfig()
	initDB()
	createFolders()
}

func initDB() {
	lo.Must0(db.Open(DatabasePath, ProjectPath, ProjectConfig.CipherKey))
}

func createFolders() {
	util.MkdirIfNotExists(WaitingFolder, 0)
	util.MkdirIfNotExists(BucketsFolder, 0)
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
		title := filepath.Base(ProjectPath)
		cipherkey := database.DefaultCipherKey()
		ProjectConfig = model.NewProject(title, cipherkey)
		writeProjectConfig()
		return
	}
	readProjectConfig()
}
