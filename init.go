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
	BucketsFolderName = "buckets"
)

var (
	db                = new(database.DB)
	ProjectPath       = filepath.Dir(util.GetExePath())
	ProjectConfig     *model.Project
	ProjectConfigPath = filepath.Join(ProjectPath, ProjectTOML)
	DatabasePath      = filepath.Join(ProjectPath, DatabaseFileName)
	BucketsFolder     = filepath.Join(ProjectPath, BucketsFolderName)
)

func init() {
	initProjectConfig()
	initDB()
	createFolders()
}

func initDB() {
	lo.Must0(db.Open(DatabasePath))
}

func createFolders() {
	util.MustMkdir(BucketsFolder)
}

func readProjectConfig() {
	data := lo.Must(os.ReadFile(ProjectConfigPath))
	lo.Must0(toml.Unmarshal(data, &ProjectConfig))
}

func initProjectConfig() {
	if util.PathIsNotExist(ProjectConfigPath) {
		title := filepath.Base(ProjectPath)
		cipherkey := database.DefaultCipherKey()
		ProjectConfig = model.NewProject(title, cipherkey)
		util.WriteTOML(ProjectConfig, ProjectConfigPath)
		return
	}
	readProjectConfig()
}
