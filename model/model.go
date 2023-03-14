package model

import (
	"errors"
	"regexp"
)

const (
	RFC3339 = "2006-01-02 15:04:05Z07:00"
)

// ID 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
var IdForbidPattern = regexp.MustCompile(`[^0-9a-zA-Z._\-]`)

// Project 工程
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

	// 倉庫標題和副標題, 可使用任何語言任意字符.
	// 其中, Title 在數據庫中是 COLLATE NOCASE UNIQUE.
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

	// 容量 (最多可容納多少個文件)
	Capacity int `json:"capacity"`

	// 文件體積上限 (單位: MB)
	MaxFilesize int `json:"max_filesize"`

	// 是否加密 (在創建時決定, 不可更改) (密碼在 ProjectConfig 中統一設定)
	Encrypted bool `json:"encrypted"`
}

// CreateBucketForm 用於新建倉庫, 由前端傳給后端.
type CreateBucketForm struct {
	ID        string `json:"id" validate:"required"`
	Encrypted bool   `json:"encrypted"`
}

func NewBucket(form *CreateBucketForm) (*Bucket, error) {
	if err := checkFilename(form.ID); err != nil {
		return nil, err
	}
	b := new(Bucket)
	b.ID = form.ID
	b.Title = form.ID
	b.Capacity = 1024
	b.MaxFilesize = 1024 // MB
	b.Encrypted = form.Encrypted
	return b, nil
}

// File 文件.
// 当 adler32 没有冲突时, sha256 取 nil 值,
// 当 adler32 有冲突时, 必须同时记录 adler32 和 sha256.
// sha256 只允許空字符串重複, 有內容的值不允許重複.
// Notes 與 Keywords 本質上是一樣的, 只是一行字符串, 用來輔助搜尋.
type File struct {
	ID       int64  `json:"id"`       // 自動數字ID
	Adler32  string `json:"adler32"`  // NOT NULL, 允許重複
	Sha256   string `json:"sha256"`   // NOT NULL, 允許重複
	BucketID string `json:"bucketid"` // Bucket.ID
	Name     string `json:"name"`     // 文件名
	Notes    string `json:"notes"`    // 備註
	Keywords string `json:"keywors"`  // 關鍵詞, 便於搜尋
	Size     int64  `json:"size"`     // length in bytes for regular files
	Type     string `json:"type"`     // 文件類型, 例: text/js, office/docx
	Like     int64  `json:"like"`     // 點贊
	CTime    string `json:"ctime"`    // RFC3339 文件入庫時間
	UTime    string `json:"utime"`    // RFC3339 文件更新時間
	Checked  string `json:"checked"`  // RFC3339 上次校驗文件完整性的時間
	Damaged  bool   `json:"damaged"`  // 上次校驗結果 (文件是否損壞)
	Deleted  bool   `json:"deleted"`  // 把文件标记为 "已删除"
}

func NewFile(root, bucketID, name string) *File {
	f := new(File)
	return f
	// TODO
}

func checkFilename(name string) error {
	if IdForbidPattern.MatchString(name) {
		return errors.New("只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)," +
			"\n不可使用空格, 请用下劃線或連字號代替空格。")
	}
	return nil
}

type CheckPwdForm struct {
	Password string `json:"password" validate:"required"`
}

type ChangePwdForm struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}
