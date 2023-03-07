package main

import (
	"github.com/ahui2016/local-buckets/model"
	"github.com/gofiber/fiber/v2"
)

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectPath})
}
