package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
)

func main() {
	app := fiber.New()

	app.Static("/", "./public")

	api := app.Group("/api")
	api.Get("/project-config", getProjectConfig) // resp: Project
	api.Get("/all-buckets", getAllBuckets)       // resp: null | Bucket[]

	log.Fatal(app.Listen(ProjectConfig.Host))
}
