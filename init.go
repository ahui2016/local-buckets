package main

import (
	"os"
	"path/filepath"

	"github.com/ahui2016/local-buckets/model"
	"github.com/ahui2016/local-buckets/util"
	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

const (
	ProjectTOML = "project.toml"
)

var (
	ProjectPath       = filepath.Dir(util.GetExePath())
	ProjectConfig     *model.Project
	ProjectConfigPath = filepath.Join(ProjectPath, ProjectTOML)
)

func init() {
	initProjectConfig()
}

func readProjectConfig() {
	data := lo.Must(os.ReadFile(ProjectConfigPath))
	lo.Must0(toml.Unmarshal(data, &ProjectConfig))
}

func initProjectConfig() {
	if util.PathIsNotExist(ProjectConfigPath) {
		title := filepath.Base(ProjectPath)
		ProjectConfig = model.NewProject(title)
		util.WriteTOML(ProjectConfig, ProjectConfigPath)
		return
	}
	readProjectConfig()
}
