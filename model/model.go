package model

type Project struct {
	Host     string `json:"host"`
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
}

type ProjectInfo struct {
	*Project
	Path string `json:"path"`
}

func NewProject(title string) *Project {
	return &Project{"127.0.0.1:3000", title, ""}
}
