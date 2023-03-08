package model

type Project struct {
	Host      string `json:"host"`
	Title     string `json:"title"`
	Subtitle  string `json:"subtitle"`
	CipherKey string `json:"cipherkey"` // 被加密的真正密鑰
}

type ProjectInfo struct {
	*Project
	Path string `json:"path"`
}

func NewProject(title string, cipherkey string) *Project {
	return &Project{"127.0.0.1:3000", title, "", cipherkey}
}

// Bucket 倉庫
type Bucket struct {
	// 倉庫 ID, 同時也是倉庫資料夾名.
	// 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
	// 注意, 在數據庫中, ID 是不分大小寫的.
	ID string `json:"id"`

	// 倉庫標題和副標題, 可使用任何語言任意字符
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

	// 容量 (最多可容納多少個文件)
	Capacity int `json:"capacity"`

	// 文件體積上限 (單位: MB)
	MaxFilesize int `json:"max_filesize"`

	// 是否加密 (在創建時決定, 不可更改) (密碼在 ProjectConfig 中統一設定)
	Encrypted bool `json:"encrypted"`
}

type ChangePwdForm struct {
	OldPassword string `json:"old_password"`
}
