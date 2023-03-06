package main

import "github.com/gofiber/fiber/v2"

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(ProjectConfig)
}
