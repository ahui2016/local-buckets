package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/ahui2016/local-buckets/database"
	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/util"
	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

type (
	Project = model.Project
	Bucket  = model.Bucket
	File    = model.File
)

const (
	ProjectTOML       = "project.toml"
	DatabaseFileName  = "project.db"
	WaitingFolderName = "waiting"
	BucketsFolderName = "buckets"
	TempFolderName    = "temp"
	TempFilesJsonName = "temp_files.json"
	PublicFolderName  = "public"
)

var (
	db                = new(database.DB)
	ProjectPath       = filepath.Dir(util.GetExePath())
	ProjectConfig     *Project
	ProjectConfigPath = filepath.Join(ProjectPath, ProjectTOML)
	DatabasePath      = filepath.Join(ProjectPath, DatabaseFileName)
	WaitingFolder     = filepath.Join(ProjectPath, WaitingFolderName)
	BucketsFolder     = filepath.Join(ProjectPath, BucketsFolderName)
	TempFolder        = filepath.Join(ProjectPath, TempFolderName)
	TempFilesJsonPath = filepath.Join(TempFolder, TempFilesJsonName)
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
		title := filepath.Base(ProjectPath)
		cipherkey := database.DefaultCipherKey()
		ProjectConfig = model.NewProject(title, cipherkey)
		writeProjectConfig()
		return
	}
	readProjectConfig()
}

func getTempFiles() (map[string]*File, error) {
	files := make(map[string]*File)
	filesJSON, err := os.ReadFile(TempFilesJsonPath)
	if err != nil {
		// 如果读取文件失败，则反回一个空的 filesInfo, 不处理错误。
		return files, nil
	}
	err = json.Unmarshal(filesJSON, &files)
	return files, err
}

func getWaitingFiles() ([]string, error) {
	files, err := util.GetRegularFiles(WaitingFolder)
	if err != nil {
		return nil, err
	}
}
