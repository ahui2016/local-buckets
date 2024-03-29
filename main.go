package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	defer db.DB.Close()
	app := fiber.New(fiber.Config{
		Immutable: true, // 以后试试删除该设定
	})

	app.Use(noCache)

	app.Static("/", PublicFolder)

	app.Get("/file/:id", previewFile)

	api := app.Group("/api", sleep)

	api.Use("/update-bucket-info", notAllowInBackup)
	api.Use("/delete-bucket", deleteBucket)
	api.Use("/create-bucket", notAllowInBackup)
	api.Use("/imported-files", notAllowInBackup)
	api.Use("/waiting-files", notAllowInBackup)
	api.Use("/upload-new-files", notAllowInBackup)
	api.Use("/import-files", notAllowInBackup)
	api.Use("/rename-waiting-file", notAllowInBackup)
	api.Use("/overwrite-file", notAllowInBackup)
	api.Use("/delete-file", notAllowInBackup)
	// api.Use("/create-new-note", notAllowInBackup)
	api.Use("/update-file-info", notAllowInBackup)
	api.Use("/move-file-to-bucket", notAllowInBackup)
	api.Use("/change-password", notAllowInBackup)

	api.Post("/update-bucket-info", updateBucketHandler)
	api.Post("/delete-bucket", deleteBucket)
	api.Post("/create-bucket", createBucket)            // resp.data: Bucket
	api.Get("/imported-files", getImportedFilesHandler) // resp.data: File[]
	api.Get("/waiting-files", getWaitingFiles)          // resp.data: File[] | ErrSameNameFiles
	api.Post("/upload-new-files", uploadNewFiles)
	api.Post("/import-files", importFiles)
	api.Post("/rename-waiting-file", renameWaitingFile)
	api.Post("/overwrite-file", overwriteFile)
	api.Post("/delete-file", deleteFile)
	api.Get("/create-new-note", createNewNote)
	api.Post("/update-file-info", updateFileInfo)      // resp.data: FilePlus
	api.Post("/move-file-to-bucket", moveFileToBucket) // resp.data: FilePlus
	api.Post("/change-password", changePassword)

	api.Use("/rebuild-thumbs", requireAdmin)
	api.Post("/rebuild-thumbs", rebuildThumbsHandler)

	api.Get("/waiting-folder", getWaitingFolder)   // resp.data: TextMsg
	api.Get("/auto-get-keywords", autoGetKeywords) // resp.data: null | string[]
	api.Get("/auto-get-buckets", autoGetBuckets)   // resp.data: null | BucketStatus[]
	api.Post("/get-bucket", getBucketHandler)
	api.Post("/download-file", downloadFile)
	api.Post("/download-small-pic", downloadSmallPic)
	api.Post("/set-export", setExportHandler)
	api.Post("/file-info", getFileByID)    // resp.data: FilePlus
	api.Post("/files", getFilesHandler)    // resp.data: FilePlus[]
	api.Post("/pics", getPicsHandler)      // resp.data: FilePlus[]
	api.Post("/search-files", searchFiles) // resp.data: FilePlus[]
	api.Post("/search-pics", searchPics)   // resp.data: FilePlus[]

	api.Post("/create-bk-proj", createBKProjHandler)
	api.Post("/delete-bk-proj", deleteBKProjHandler)
	api.Get("/project-status", getProjectStatus)   // resp.data: ProjectStatus
	api.Post("/bk-project-status", getBKProjStat)  // resp.data: ProjectStatus
	api.Post("/check-now", checkNow)               // resp.data: ProjectStatus
	api.Get("/damaged-files", damagedFilesHandler) // resp.data: FilePlus[]
	api.Post("/repair-files", repairFilesHandler)
	api.Post("/sync-backup", syncBackup)

	api.Get("/login-status", getLoginStatus) // resp.data: OneTextForm
	api.Post("/admin-login", adminLogin)
	api.Get("/logout", logoutHandler)

	log.Fatal(app.Listen(ProjectConfig.Host))
}
