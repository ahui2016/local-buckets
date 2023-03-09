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
	f := new(model.CheckPwdForm)
	if err := c.BodyParser(f); err != nil {
		return err
	}
	_, err := db.SetAESGCM(f.Password)
	return err
}

func changePassword(c *fiber.Ctx) error {
	f := new(model.ChangePwdForm)
	if err := c.BodyParser(f); err != nil {
		return err
	}
	cipherKey, err := db.ChangePassword(f.OldPassword, f.NewPassword)
	if err != nil {
		return err
	}
	ProjectConfig.CipherKey = cipherKey
	writeProjectConfig()
	return nil
}

func getAllBuckets(c *fiber.Ctx) error {
	buckets, err := db.GetAllBuckets()
	if err != nil {
		return err
	}
	return c.JSON(buckets)
}
