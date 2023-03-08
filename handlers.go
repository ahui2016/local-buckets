package main

import (
	"net/http"

	"github.com/ahui2016/local-buckets/model"
	"github.com/gofiber/fiber/v2"
)

const OK = http.StatusOK

// Text 用于向前端返回一个简单的文本消息。
// 为了保持一致性，总是向前端返回 JSON, 因此即使是简单的文本消息也使用 JSON.
type Text struct {
	Message string `json:"message"`
}

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectPath})
}

func getAllBuckets(c *fiber.Ctx) error {
	buckets, err := db.GetAllBuckets()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}
