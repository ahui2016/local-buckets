package model

type Project struct {
	Host     string
	Title    string
	Subtitle string
}

func NewProject(title string) *Project {
	return &Project{"127.0.0.1:3000", title, ""}
}
