package main

import (
	"net/http"

	"github.com/ahui2016/local-buckets/model"
	"github.com/gofiber/fiber/v2"
)

const OK = http.StatusOK

// TextMsg 用于向前端返回一个简单的文本消息。
type TextMsg struct {
	Text string `json:"text"`
}

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectPath})
}

func checkPassword(c *fiber.Ctx) error {
	f := new(model.ChangePwdForm)
	if err := c.BodyParser(f); err != nil {
		return err
	}
	return db.SetAESGCM(f.OldPassword)
}

func getAllBuckets(c *fiber.Ctx) error {
	buckets, err := db.GetAllBuckets()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}
