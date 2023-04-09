package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	defer db.DB.Close()
	app := fiber.New()

	app.Static("/", PublicFolder)

	app.Get("/file/:id", previewFile)

	api := app.Group("/api", sleep)
	api.Get("/project-info", getProjectInfo)            // resp.data: ProjectInfo
	api.Get("/auto-get-buckets", autoGetBuckets)        // resp.data: null | Bucket[]
	api.Post("/create-bucket", createBucket)            // resp.data: Bucket
	api.Get("/waiting-folder", getWaitingFolder)        // resp.data: TextMsg
	api.Get("/imported-files", getImportedFilesHandler) // resp.data: File[]
	api.Get("/waiting-files", getWaitingFiles)          // resp.data: File[] | ErrSameNameFiles
	api.Post("/upload-new-files", uploadNewFiles)
	api.Post("/import-files", importFiles)
	api.Post("/download-file", downloadFile)
	api.Post("/rename-waiting-file", renameWaitingFile)
	api.Post("/overwrite-file", overwriteFile)
	api.Post("/delete-file", deleteFile)

	api.Get("/recent-files", getRecentFiles)           // resp.data: File[]
	api.Post("/file-info", getFileByID)                // resp.data: FilePlus
	api.Post("/update-file-info", updateFileInfo)      // resp.data: FilePlus
	api.Post("/move-file-to-bucket", moveFileToBucket) // resp.data: FilePlus

	api.Post("/create-bk-proj", createBKProjHandler)
	api.Post("/delete-bk-proj", deleteBKProjHandler)
	api.Get("/project-status", getProjectStatus)  // resp.data: ProjectStatus
	api.Post("/bk-project-status", getBKProjStat) // resp.data: ProjectStatus
	api.Post("/sync-backup", syncBackup)

	api.Get("/login-status", getLoginStatus) // resp.data: OneTextForm
	api.Get("/logout", logoutHandler)
	api.Post("/change-password", changePassword)
	api.Post("/admin-login", adminLogin)

	log.Fatal(app.Listen(ProjectConfig.Host))
}
