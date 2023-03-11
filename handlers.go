package main

import (
	"net/http"

	"github.com/ahui2016/local-buckets/model"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

const OK = http.StatusOK

var validate = validator.New()

// TextMsg 用于向前端返回一个简单的文本消息。
type TextMsg struct {
	Text string `json:"text"`
}

func parseValidate(form any, c *fiber.Ctx) error {
	if err := c.BodyParser(form); err != nil {
		return err
	}
	return validate.Struct(form)
}

func getProjectConfig(c *fiber.Ctx) error {
	return c.JSON(model.ProjectInfo{Project: ProjectConfig, Path: ProjectPath})
}

func checkPassword(c *fiber.Ctx) error {
	f := new(model.CheckPwdForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	_, err := db.SetAESGCM(f.Password)
	return err
}

func changePassword(c *fiber.Ctx) error {
	f := new(model.ChangePwdForm)
	if err := parseValidate(f, c); err != nil {
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

func createBucket(c *fiber.Ctx) error {
	f := new(model.CreateBucketForm)
	if err := parseValidate(f, c); err != nil {
		return err
	}
	bucket, err := db.InsertBucket(f)
	if err != nil {
		return err
	}
	createBucketFolder(f.ID)
	return c.JSON(bucket)
}
