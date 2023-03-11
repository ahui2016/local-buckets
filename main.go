package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Static("/", PublicFolder)

	api := app.Group("/api")
	api.Get("/project-config", getProjectConfig) // resp.data: Project
	api.Get("/all-buckets", getAllBuckets)       // resp.data: null | Bucket[]
	api.Post("/create-bucket", createBucket)     // resp.data: Bucket
	api.Post("/change-password", changePassword)

	log.Fatal(app.Listen(ProjectConfig.Host))
}
