package handlers

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"go-nx-admin/internal/config"
	"go-nx-admin/internal/models"
	"go-nx-admin/internal/storage"
)

const presignTTL = 15 * time.Minute

func mediaStorage() (*storage.Registry, error) {
	if r := storage.Default(); r != nil {
		return r, nil
	}
	return nil, fmt.Errorf("storage not initialised")
}

// withURL resolves a browser address for the record. Local files keep the
// stored URL (served by the /uploads route); remote drivers are asked, because
// presigned links expire and must not be persisted.
func withURL(c *fiber.Ctx, reg *storage.Registry, m models.Media) models.Media {
	if m.Storage == "" || m.Storage == "local" {
		return m
	}
	drv, err := reg.For(m.Storage)
	if err != nil {
		return m
	}
	if u, err := drv.URL(c.UserContext(), m.Path, presignTTL); err == nil {
		m.URL = u
	}
	return m
}

func UploadMedia(db *gorm.DB) fiber.Handler {
	return func(c *fiber.Ctx) error {
		reg, err := mediaStorage()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "storage unavailable"})
		}
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"message": "file is required"})
		}
		if file.Size > config.AppConfig.Upload.MaxSize {
			return c.Status(400).JSON(fiber.Map{"message": "file too large"})
		}

		src, err := file.Open()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "cannot open uploaded file"})
		}
		defer src.Close()

		head := make([]byte, 512)
		n, _ := io.ReadFull(src, head)
		ext, mimeType, ok := checkUpload(file.Filename, head[:n])
		if !ok {
			return c.Status(400).JSON(fiber.Map{"message": "file type not allowed"})
		}
		if _, err := src.Seek(0, io.SeekStart); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "cannot read uploaded file"})
		}

		ext = "." + ext
		now := time.Now()
		newName := uuid.New().String() + ext
		key := fmt.Sprintf("uploads/%s/%s/%s/%s", now.Format("2006"), now.Format("01"), now.Format("02"), newName)

		drv := reg.Active()
		if err := drv.Put(c.UserContext(), key, src, file.Size, mimeType); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "write file error"})
		}

		mediaType := "file"
		if strings.HasPrefix(mimeType, "image/") {
			mediaType = "image"
		} else if strings.HasPrefix(mimeType, "video/") {
			mediaType = "video"
		}

		userID, _ := c.Locals("user_id").(uint)
		media := models.Media{
			OriginalName: file.Filename,
			FileName:     newName,
			Path:         key,
			URL:          "/" + key,
			Storage:      drv.Name(),
			Ext:          ext,
			MimeType:     mimeType,
			Size:         file.Size,
			Type:         mediaType,
			CreatedBy:    userID,
		}
		if err := db.Create(&media).Error; err != nil {
			// Do not leave an orphaned object behind a failed record.
			drv.Delete(c.UserContext(), key)
			return c.Status(500).JSON(fiber.Map{"message": "save record error"})
		}
		media = withURL(c, reg, media)

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
		reg, err := mediaStorage()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "storage unavailable"})
		}
		page, pageSize, offset := pagination(c)
		mediaType := c.Query("type", "")

		query := db.Model(&models.Media{})
		if mediaType != "" {
			query = query.Where("type = ?", mediaType)
		}

		var total int64
		if err := query.Count(&total).Error; err != nil {
			return dbError(c, err)
		}

		var items []models.Media
		if err := query.Order("created_at desc").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
			return dbError(c, err)
		}
		for i := range items {
			items[i] = withURL(c, reg, items[i])
		}

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
		reg, err := mediaStorage()
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "storage unavailable"})
		}
		id, _ := strconv.ParseUint(c.Params("id"), 10, 64)
		var media models.Media
		if err := db.First(&media, id).Error; err != nil {
			return c.Status(404).JSON(fiber.Map{"message": "media not found"})
		}
		drv, err := reg.For(media.Storage)
		if err != nil {
			return internalError(c, err)
		}
		// Remove the object first: if that fails the record stays, so the file
		// is never orphaned and the delete can be retried.
		if err := drv.Delete(c.UserContext(), media.Path); err != nil {
			return c.Status(500).JSON(fiber.Map{"message": "delete file error"})
		}
		db.Delete(&media)
		return c.JSON(fiber.Map{"message": "ok"})
	}
}
