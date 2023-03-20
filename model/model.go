package model

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ahui2016/local-buckets/util"
	"github.com/samber/lo"
)

const (
	RFC3339 = "2006-01-02 15:04:05Z07:00"
)

// ID 只能使用 0-9, a-z, A-Z, _(下劃線), -(連字號), .(點)
var IdForbidPattern = regexp.MustCompile(`[^0-9a-zA-Z._\-]`)

// Project 專案
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

	// 容量 (最多可容納多少個檔案)
	Capacity int64 `json:"capacity"`

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
	b.Encrypted = form.Encrypted
	return b, nil
}

// File 檔案.
// Notes 與 Keywords 本質上是一樣的, 只是一行字符串, 用來輔助搜尋.
type File struct {
	ID       int64  `json:"id"`       // 自動數字ID
	Checksum string `json:"checksum"` // NOT NULL UNIQUE
	BucketID string `json:"bucketid"` // Bucket.ID
	Name     string `json:"name"`     // 檔案名
	Notes    string `json:"notes"`    // 備註
	Keywords string `json:"keywors"`  // 關鍵詞, 便於搜尋
	Size     int64  `json:"size"`     // length in bytes for regular files
	Type     string `json:"type"`     // 檔案類型, 例: text/js, office/docx
	Like     int64  `json:"like"`     // 點贊
	CTime    string `json:"ctime"`    // RFC3339 檔案入庫時間
	UTime    string `json:"utime"`    // RFC3339 檔案更新時間
	Checked  string `json:"checked"`  // RFC3339 上次校驗檔案完整性的時間
	Damaged  bool   `json:"damaged"`  // 上次校驗結果 (檔案是否損壞)
	Deleted  bool   `json:"deleted"`  // 把檔案标记为 "已删除"
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
	f.BucketID = ""
	f.Name = basename
	f.Notes = ""
	f.Keywords = ""
	f.Size = info.Size()
	f.Type = typeByFilename(basename)
	f.Like = 0
	f.CTime = now
	f.UTime = now
	f.Checked = now
	f.Damaged = false
	f.Deleted = false
	return f, nil
}

// NewFile 根据 root, bucketID, basename 生成新檔案,
// 其中 root 是專案根目录.
func NewFile(root, bucketID, basename string) (*File, error) {
	now := Now()
	filePath := filepath.Join(root, bucketID, basename)
	info, err := os.Lstat(filePath)
	if err != nil {
		return nil, err
	}
	checksum, err := util.FileSum512(filePath)
	if err != nil {
		return nil, err
	}
	f := new(File)
	f.Checksum = checksum
	f.BucketID = bucketID
	f.Name = basename
	f.Notes = ""
	f.Keywords = ""
	f.Size = info.Size()
	f.Type = typeByFilename(basename)
	f.Like = 0
	f.CTime = now
	f.UTime = now
	f.Checked = now
	f.Damaged = false
	f.Deleted = false
	return f, nil
}

func Now() string {
	return time.Now().Format(RFC3339)
}

func typeByFilename(filename string) (filetype string) {
	ext := filepath.Ext(filename)
	filetype = GetMIME(ext)
	switch ext {
	case "zip", "rar", "7z", "gz", "tar", "bz", "bz2", "xz":
		filetype = "compressed/" + ext
	case "md", "xml", "html", "xhtml", "htm", "yaml", "js", "ts", "go", "py", "cs", "dart", "rb", "c", "h", "cpp", "rs":
		filetype = "text/" + ext
	case "doc", "docx", "ppt", "pptx", "rtf", "xls", "xlsx":
		filetype = "office/" + ext
	case "epub", "pdf", "mobi", "azw", "azw3", "djvu":
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

type CheckPwdForm struct {
	Password string `json:"password" validate:"required"`
}

type ChangePwdForm struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required"`
}

type UploadToBucketForm struct {
	BucketID string `json:"bucketid" validate:"required"`
}

type MovedFile struct {
	Src string
	Dst string
}

// Move calls os.Rename, moves the file from Src to Dst.
func (m MovedFile) Move() error {
	return os.Rename(m.Src, m.Dst)
}

// Rollback moves the file from Dst back to Src.
func (m MovedFile) Rollback() error {
	return os.Rename(m.Dst, m.Src)
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
	return "同名檔案已存在(檔案名稱不分大小寫): " + e.File.Name
}

// GetMIME returns the content-type of a file extension.
// https://github.com/gofiber/fiber/blob/master/utils/http.go (edited).
func GetMIME(extension string) (mime string) {
	const MIMEOctetStream = "application/octet-stream"
	extension = strings.ToLower(extension)

	if len(extension) == 0 {
		return
	}
	if extension[0] == '.' {
		extension = extension[1:]
	}
	mime = mimeExtensions[extension]
	if len(mime) == 0 {
		return MIMEOctetStream
	}
	return mime
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
