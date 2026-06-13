package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/models"
)

func UploadMedia(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "file is required"})
		}
		if file.Size > config.AppConfig.Upload.MaxSize {
			return c.Status(400).JSON(fiber.Map{"message": "file too large"})
		}

		ext := strings.ToLower(filepath.Ext(file.Filename))
		now := time.Now()
		relDir := fmt.Sprintf("uploads/%s/%s/%s", now.Format("2006"), now.Format("01"), now.Format("02"))
		absDir := filepath.Join(config.AppConfig.Upload.Dir, relDir)
		os.MkdirAll(absDir, 0755)

		newName := uuid.New().String() + ext
		dst := filepath.Join(absDir, newName)

		src, err := file.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "cannot open uploaded file"})
		}
		defer src.Close()

		out, err := os.Create(dst)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "cannot create file"})
		}
		defer out.Close()

		if _, err := io.Copy(out, src); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "write file error"})
		}

		mimeType := file.Header.Get("Content-Type")
		mediaType := "file"
		if strings.HasPrefix(mimeType, "image/") {
			mediaType = "image"
		} else if strings.HasPrefix(mimeType, "video/") {
			mediaType = "video"
		}

		relPath := relDir + "/" + newName
		userID, _ := c.Locals("user_id").(uint)
		media := models.Media{
			OriginalName: file.Filename,
			FileName:     newName,
			Path:         relPath,
			URL:          "/" + relPath,
			Ext:          ext,
			MimeType:     mimeType,
			Size:         file.Size,
			Type:         mediaType,
			CreatedBy:    userID,
		}
		db.Create(&media)

		return c.JSON(fiber.Map{
			"id":            media.ID,
			"url":           media.URL,
			"original_name": media.OriginalName,
			"size":          media.Size,
			"type":          media.Type,
		})
	}
}

func ListMedia(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		page, _ := strconv.Atoi(c.Query("page", "1"))
		pageSize, _ := strconv.Atoi(c.Query("page_size", "20"))
		mediaType := c.Query("type", "")

		query := db.Model(&models.Media{})
		if mediaType != "" {
			query = query.Where("type = ?", mediaType)
		}

		var total int64
		query.Count(&total)

		var items []models.Media
		query.Order("created_at desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items)

		return c.JSON(fiber.Map{
			"data":      items,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

func DeleteMedia(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var media models.Media
		if err := db.First(&media, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "media not found"})
		}
		os.Remove(media.Path)
		db.Delete(&media)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}
