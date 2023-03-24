package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Static("/", PublicFolder)

	api := app.Group("/api", sleep)
	api.Get("/project-config", getProjectConfig) // resp.data: Project
	api.Get("/all-buckets", getAllBuckets)       // resp.data: null | Bucket[]
	api.Post("/create-bucket", createBucket)     // resp.data: Bucket
	api.Post("/change-password", changePassword)
	api.Get("/waiting-folder", getWaitingFolder) // resp.data: TextMsg
	api.Get("/waiting-files", getWaitingFiles)   // resp.data: File[] | ErrSameNameFiles
	api.Post("/upload-new-files", uploadNewFiles)
	api.Post("/rename-waiting-file", renameWaitingFile)
	api.Post("/overwrite-file", overwriteFile)

	api.Get("/recent-files", getRecentFiles) // resp.data: File[]
	api.Post("/update-file-info", updateFileInfo)

	log.Fatal(app.Listen(ProjectConfig.Host))
}
