package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	defer db.DB.Close()
	app := fiber.New()

	app.Static("/", PublicFolder)

	api := app.Group("/api", sleep)
	api.Get("/project-info", getProjectInfo)     // resp.data: ProjectInfo
	api.Get("/auto-get-buckets", autoGetBuckets) // resp.data: null | Bucket[]
	api.Post("/create-bucket", createBucket)     // resp.data: Bucket
	api.Post("/change-password", changePassword)
	api.Get("/waiting-folder", getWaitingFolder) // resp.data: TextMsg
	api.Get("/waiting-files", getWaitingFiles)   // resp.data: File[] | ErrSameNameFiles
	api.Post("/upload-new-files", uploadNewFiles)
	api.Post("/rename-waiting-file", renameWaitingFile)
	api.Post("/overwrite-file", overwriteFile)

	api.Get("/recent-files", getRecentFiles) // resp.data: File[]
	api.Post("/file-info", getFileByID)      // resp.data: File
	api.Post("/update-file-info", updateFileInfo)
	api.Post("/move-file-to-bucket", moveFileToBucket)

	api.Post("/create-bk-proj", createBKProjHandler)
	api.Post("/delete-bk-proj", deleteBKProjHandler)
	api.Get("/project-status", getProjectStatus)  // resp.data: ProjectStatus
	api.Post("/bk-project-status", getBKProjStat) // resp.data: ProjectStatus
	api.Post("/sync-backup", syncBackup)

	log.Fatal(app.Listen(ProjectConfig.Host))
}
