package model

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ahui2016/local-buckets/util"
	"github.com/pelletier/go-toml/v2"
	"github.com/samber/lo"
)

const (
	RFC3339         = "2006-01-02 15:04:05Z07:00"
	MIMEOctetStream = "application/octet-stream"
)

// ID 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
var IdForbidPattern = regexp.MustCompile(`[^0-9a-zA-Z._\-]`)

// Project 專案
type Project struct {
	Host             string   `json:"host"`
	Title            string   `json:"title"`
	Subtitle         string   `json:"subtitle"`
	CipherKey        string   `json:"cipherkey"` // 被加密的真正密鑰
	ApiDelay         int64    `json:"api_delay"` // 後端 API 延遲, 單位: 毫秒
	RecentFilesLimit int64    `json:"recent_files_limit"`
	CheckInterval    int64    `jso:"check_interval"` // 檢查周期, 單位: day
	IsBackup         bool     `json:"is_backup"`
	BackupProjects   []string `json:"backup_projects"`
	LastBackupAt     string   `json:"last_backup_at"`  // RFC3339
	DownloadExport   bool     `json:"download_export"` // 下載時導出
	MarkdownStyle    string   `json:"markdown_style"`
}

func NewProject(title string, cipherkey string) *Project {
	return &Project{
		Host:             "127.0.0.1:3000",
		Title:            title,
		CipherKey:        cipherkey,
		RecentFilesLimit: 100,
		CheckInterval:    30,
	}
}

type ProjectStatus struct {
	*Project
	Root              string // 专案根目录
	TotalSize         int64  // 全部檔案體積合計
	FilesCount        int64  // 檔案數量合計
	WaitingCheckCount int64  // 待檢查檔案數量合計
	DamagedCount      int64  // 損壞檔案數量合計
}

// Bucket 倉庫
type Bucket struct {
	// 自增數字ID
	ID int64 `json:"id"`

	// 倉庫資料夾名, 不分大小寫.
	// 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
	Name string `json:"name"`

	// 倉庫標題和副標題, 可使用任何語言任意字符.
	// 其中, Title 在數據庫中是 COLLATE NOCASE UNIQUE.
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`

	// 是否加密 (在創建時決定, 不可更改) (密碼在 ProjectConfig 中統一設定)
	Encrypted bool `json:"encrypted"`
}

type BucketStatus struct {
	*Bucket
	TotalSize  int64
	FilesCount int64
}

// CreateBucketForm 用於新建倉庫, 由前端傳給后端.
type CreateBucketForm struct {
	Name      string `json:"name" validate:"required"`
	Encrypted bool   `json:"encrypted"`
}

func NewBucket(form *CreateBucketForm) (*Bucket, error) {
	if err := checkFilename(form.Name); err != nil {
		return nil, err
	}
	b := new(Bucket)
	b.Name = form.Name
	b.Title = form.Name
	b.Encrypted = form.Encrypted
	return b, nil
}

type FileExportImport struct {
	BucketName string
	Notes      string
	Keywords   string
	Like       int64
	CTime      string
	UTime      string
}

// File 檔案.
// Notes 與 Keywords 本質上是一樣的, 只是一行字符串, 用來輔助搜尋.
type File struct {
	ID         int64  `json:"id"`          // 自增數字ID
	Checksum   string `json:"checksum"`    // NOT NULL UNIQUE
	BucketName string `json:"bucket_name"` // Bucket.Name
	Name       string `json:"name"`        // 檔案名
	Notes      string `json:"notes"`       // 備註
	Keywords   string `json:"keywords"`    // 關鍵詞, 便於搜尋
	Size       int64  `json:"size"`        // length in bytes for regular files
	Type       string `json:"type"`        // 檔案類型, 例: text/js, office/docx
	Like       int64  `json:"like"`        // 點贊
	CTime      string `json:"ctime"`       // RFC3339 檔案入庫時間
	UTime      string `json:"utime"`       // RFC3339 檔案更新時間
	Checked    string `json:"checked"`     // RFC3339 上次校驗檔案完整性的時間
	Damaged    bool   `json:"damaged"`     // 上次校驗結果 (檔案是否損壞)
	Deleted    bool   `json:"deleted"`     // 把檔案标记为 "已删除"
}

func (f *File) Rename(name string) {
	if f.Name == name {
		return
	}
	f.Name = name
	f.Type = typeByFilename(name)
}

func (f *File) ImportFrom(f2 FileExportImport) {
	f.BucketName = f2.BucketName
	f.Notes = f2.Notes
	f.Keywords = f2.Keywords
	f.Like = f2.Like
	f.CTime = f2.CTime
	f.UTime = f2.UTime
}

func (f *File) IsImage() bool {
	return strings.HasPrefix(f.Type, "image")
}

func (f *File) IsText() bool {
	return strings.HasPrefix(f.Type, "text")
}

func (f *File) IsPDF() bool {
	return f.Type == "application/pdf"
}

// 添加 音频/视频 支持
func (f *File) CanBePreviewed() bool {
	return f.IsImage() || f.IsText() || f.IsPDF()
}

func ExportFileFrom(f File) FileExportImport {
	return FileExportImport{
		f.BucketName,
		f.Notes,
		f.Keywords,
		f.Like,
		f.CTime,
		f.UTime,
	}
}

func ImportFileFrom(tomlPath string) (imported FileExportImport, err error) {
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return
	}
	err = toml.Unmarshal(data, &imported)
	return
}

// FilePlus 檔案以及更多資訊.
type FilePlus struct {
	File
	Encrypted bool `json:"encrypted"`
}

// NewWaitingFile 根据 filePath 生成新檔案,
// 其中 filePath 是等待上传的檔案的路径.
func NewWaitingFile(filePath string) (*File, error) {
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	checksum, err := util.FileSum512(filePath)
	if err != nil {
		return nil, err
	}
	basename := filepath.Base(filePath)
	now := Now()
	f := new(File)
	f.Checksum = checksum
	f.Name = basename
	f.Size = info.Size()
	f.Type = typeByFilename(basename)
	f.CTime = now
	f.UTime = now
	f.Checked = now
	return f, nil
}

// Now return time.Now().Format(RFC3339)
func Now() string {
	return time.Now().Format(RFC3339)
}

// https://github.com/gofiber/fiber/blob/master/utils/http.go (edited).
func typeByFilename(filename string) (filetype string) {
	ext := filepath.Ext(filename)
	ext = strings.ToLower(ext)
	if len(ext) == 0 {
		return MIMEOctetStream
	}
	if ext[0] == '.' {
		ext = ext[1:]
	}
	filetype = mimeExtensions[ext]
	if len(filetype) == 0 {
		filetype = MIMEOctetStream
	}

	switch ext {
	case "zip", "rar", "7z", "gz", "tar", "bz", "bz2", "xz":
		filetype = "compressed/" + ext
	case "md", "json", "xml", "html", "xhtml", "htm", "atom", "rss", "yaml",
		"js", "ts", "go", "py", "cs", "dart", "rb", "c", "h", "cpp", "rs":
		filetype = "text/" + ext
	case "doc", "docx", "ppt", "pptx", "rtf", "xls", "xlsx":
		filetype = "office/" + ext
	case "epub", "mobi", "azw", "azw3", "djvu":
		filetype = "ebook/" + ext
	}
	return filetype
}

// FilesToString 把多个檔案转换为多个檔案的 ID 的字符串.
func FilesToString(files []File) string {
	ids := lo.Map(files, func(x File, _ int) string {
		return strconv.Itoa(int(x.ID))
	})
	return strings.Join(ids, ", ")
}

func checkFilename(name string) error {
	if IdForbidPattern.MatchString(name) {
		return errors.New("只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)," +
			"\n不可使用空格, 请用下劃線或連字號代替空格。")
	}
	return nil
}

type OneTextForm struct {
	Text string `json:"text" validate:"required"`
}

type FileIdForm struct {
	ID int64 `json:"id" params:"id" validate:"required,gt=0"`
}

type FileIdRangeForm struct {
	Start int64 `json:"start" params:"start" validate:"required,gt=0"`
	End   int64 `json:"end" params:"end" validate:"required,gt=0"`
}

type ChangePwdForm struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

type RenameWaitingFileForm struct {
	OldName string `json:"old_name" validate:"required"`
	NewName string `json:"new_name" validate:"required"`
}

type UpdateFileInfoForm struct {
	ID       int64  `json:"id" validate:"required,gt=0"`
	Name     string `json:"name" validate:"required"`
	Notes    string `json:"notes"`
	Keywords string `json:"keywords"`
	Like     int64  `json:"like"`
	CTime    string `json:"ctime" validate:"required"`
	UTime    string `json:"utime"`
}

type MoveFileToBucketForm struct {
	FileID     int64  `json:"file_id" validate:"required,gt=0"`
	BucketName string `json:"bucket_name" validate:"required"`
}

type MovedFile struct {
	Src string
	Dst string
}

// Move calls os.Rename, moves the file from Src to Dst.
func (m *MovedFile) Move() error {
	err1 := os.Rename(m.Src, m.Dst)
	err2 := util.LockFile(m.Dst)
	return util.WrapErrors(err1, err2)
}

// Rollback moves the file from Dst back to Src.
func (m *MovedFile) Rollback() error {
	err1 := os.Rename(m.Dst, m.Src)
	err2 := util.LockFile(m.Src)
	return util.WrapErrors(err1, err2)
}

type ErrSameNameFiles struct {
	File    File   `json:"file"`
	ErrType string `json:"errType"`
}

func NewErrSameNameFiles(file File) ErrSameNameFiles {
	return ErrSameNameFiles{
		File:    file,
		ErrType: "ErrSameNameFiles",
	}
}

func (e ErrSameNameFiles) Error() string {
	return fmt.Sprintf(
		"倉庫中已有同名檔案(檔案名稱不分大小寫): %s/%s", e.File.BucketName, e.File.Name)
}

// MIME types were copied from
// https://github.com/gofiber/fiber/blob/master/utils/http.go
// https://github.com/nginx/nginx/blob/master/conf/mime.types
var mimeExtensions = map[string]string{
	"html":    "text/html",
	"htm":     "text/html",
	"shtml":   "text/html",
	"css":     "text/css",
	"xml":     "application/xml",
	"gif":     "image/gif",
	"jpeg":    "image/jpeg",
	"jpg":     "image/jpeg",
	"js":      "text/javascript",
	"atom":    "application/atom+xml",
	"rss":     "application/rss+xml",
	"mml":     "text/mathml",
	"txt":     "text/plain",
	"jad":     "text/vnd.sun.j2me.app-descriptor",
	"wml":     "text/vnd.wap.wml",
	"htc":     "text/x-component",
	"avif":    "image/avif",
	"png":     "image/png",
	"svg":     "image/svg+xml",
	"svgz":    "image/svg+xml",
	"tif":     "image/tiff",
	"tiff":    "image/tiff",
	"wbmp":    "image/vnd.wap.wbmp",
	"webp":    "image/webp",
	"ico":     "image/x-icon",
	"jng":     "image/x-jng",
	"bmp":     "image/x-ms-bmp",
	"woff":    "font/woff",
	"woff2":   "font/woff2",
	"jar":     "application/java-archive",
	"war":     "application/java-archive",
	"ear":     "application/java-archive",
	"json":    "application/json",
	"hqx":     "application/mac-binhex40",
	"doc":     "application/msword",
	"pdf":     "application/pdf",
	"ps":      "application/postscript",
	"eps":     "application/postscript",
	"ai":      "application/postscript",
	"rtf":     "application/rtf",
	"m3u8":    "application/vnd.apple.mpegurl",
	"kml":     "application/vnd.google-earth.kml+xml",
	"kmz":     "application/vnd.google-earth.kmz",
	"xls":     "application/vnd.ms-excel",
	"eot":     "application/vnd.ms-fontobject",
	"ppt":     "application/vnd.ms-powerpoint",
	"odg":     "application/vnd.oasis.opendocument.graphics",
	"odp":     "application/vnd.oasis.opendocument.presentation",
	"ods":     "application/vnd.oasis.opendocument.spreadsheet",
	"odt":     "application/vnd.oasis.opendocument.text",
	"pptx":    "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	"xlsx":    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"docx":    "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"wmlc":    "application/vnd.wap.wmlc",
	"wasm":    "application/wasm",
	"7z":      "application/x-7z-compressed",
	"cco":     "application/x-cocoa",
	"jardiff": "application/x-java-archive-diff",
	"jnlp":    "application/x-java-jnlp-file",
	"run":     "application/x-makeself",
	"pl":      "application/x-perl",
	"pm":      "application/x-perl",
	"prc":     "application/x-pilot",
	"pdb":     "application/x-pilot",
	"rar":     "application/x-rar-compressed",
	"rpm":     "application/x-redhat-package-manager",
	"sea":     "application/x-sea",
	"swf":     "application/x-shockwave-flash",
	"sit":     "application/x-stuffit",
	"tcl":     "application/x-tcl",
	"tk":      "application/x-tcl",
	"der":     "application/x-x509-ca-cert",
	"pem":     "application/x-x509-ca-cert",
	"crt":     "application/x-x509-ca-cert",
	"xpi":     "application/x-xpinstall",
	"xhtml":   "application/xhtml+xml",
	"xspf":    "application/xspf+xml",
	"zip":     "application/zip",
	"bin":     "application/octet-stream",
	"exe":     "application/octet-stream",
	"dll":     "application/octet-stream",
	"deb":     "application/octet-stream",
	"dmg":     "application/octet-stream",
	"iso":     "application/octet-stream",
	"img":     "application/octet-stream",
	"msi":     "application/octet-stream",
	"msp":     "application/octet-stream",
	"msm":     "application/octet-stream",
	"mid":     "audio/midi",
	"midi":    "audio/midi",
	"kar":     "audio/midi",
	"mp3":     "audio/mpeg",
	"ogg":     "audio/ogg",
	"m4a":     "audio/x-m4a",
	"ra":      "audio/x-realaudio",
	"3gpp":    "video/3gpp",
	"3gp":     "video/3gpp",
	"ts":      "video/mp2t",
	"mp4":     "video/mp4",
	"mpeg":    "video/mpeg",
	"mpg":     "video/mpeg",
	"mov":     "video/quicktime",
	"webm":    "video/webm",
	"flv":     "video/x-flv",
	"m4v":     "video/x-m4v",
	"mng":     "video/x-mng",
	"asx":     "video/x-ms-asf",
	"asf":     "video/x-ms-asf",
	"wmv":     "video/x-ms-wmv",
	"avi":     "video/x-msvideo",
}
