package routes

import (
	"strings"

	"github.com/Open-Source-Life/AxolotlDrive/middlewares"
	"github.com/Open-Source-Life/AxolotlDrive/services"
	publicfiles "github.com/Open-Source-Life/AxolotlDrive/services/public_files"
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

func SetupRoutes(app *fiber.Router, db *gorm.DB) {
	(*app).Use(middlewares.Recovery())
	(*app).Use(middlewares.Logger())
	(*app).Use(middlewares.CORS())
	(*app).Use(middlewares.RateLimiter())
	(*app).Get("/healthz", func(c *fiber.Ctx) error {
		return services.HealthCheck(c)
	})

	wsHub := publicfiles.NewWebSocketHub()
	go wsHub.Run()

	publicFilesService := publicfiles.NewPublicFilesService("data/public", wsHub)

	(*app).Get("/files", func(c *fiber.Ctx) error {
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 50)
		items, errResp := publicFilesService.ListItemsRoot(page, limit)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(items)
	})

	(*app).Get("/files/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 50)
		items, errResp := publicFilesService.SearchItems(query, page, limit)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(items)
	})

	// Specific routes first
	(*app).Get("/files/download/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		data, errResp := publicFilesService.DownloadItem(path)
		if errResp != nil {
			return c.Status(fiber.StatusNotFound).JSON(errResp)
		}
		return c.Send(data)
	})

	(*app).Get("/files/download-folder/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		files, errResp := publicFilesService.DownloadFolder(path)
		if errResp != nil {
			return c.Status(fiber.StatusNotFound).JSON(errResp)
		}
		return c.JSON(files)
	})

	(*app).Get("/files/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		page := c.QueryInt("page", 1)
		limit := c.QueryInt("limit", 50)
		items, errResp := publicFilesService.ListItems(path, page, limit)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(items)
	})

	(*app).Post("/files/upload/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "No file provided"})
		}
		f, _ := file.Open()
		defer f.Close()
		result, errResp := publicFilesService.UploadFile(path, f)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/mkdir/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		result, errResp := publicFilesService.CreateFolder(path)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/create-file/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		result, errResp := publicFilesService.CreateFile(path)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Delete("/files/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		result, errResp := publicFilesService.DeleteItem(path)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Put("/files/edit/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		content := string(c.Body())
		result, errResp := publicFilesService.EditFile(path, content)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/rename", func(c *fiber.Ctx) error {
		var req struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.RenameFile(req.OldPath, req.NewPath)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/rename-folder", func(c *fiber.Ctx) error {
		var req struct {
			OldPath string `json:"old_path"`
			NewPath string `json:"new_path"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.RenameFolder(req.OldPath, req.NewPath)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/move", func(c *fiber.Ctx) error {
		var req struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.MoveFile(req.Source, req.Destination)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/move-folder", func(c *fiber.Ctx) error {
		var req struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.MoveFolder(req.Source, req.Destination)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/copy", func(c *fiber.Ctx) error {
		var req struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.CopyFile(req.Source, req.Destination)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/copy-folder", func(c *fiber.Ctx) error {
		var req struct {
			Source      string `json:"source"`
			Destination string `json:"destination"`
		}
		c.BodyParser(&req)
		result, errResp := publicFilesService.CopyFolder(req.Source, req.Destination)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Post("/files/upload-folder/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		var files map[string][]byte
		if err := c.BodyParser(&files); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
		}
		result, errResp := publicFilesService.UploadFolder(path, files)
		if errResp != nil {
			return c.Status(fiber.StatusBadRequest).JSON(errResp)
		}
		return c.JSON(result)
	})

	(*app).Get("/ws/public_files", websocket.New(wsHub.HandleConnection))
}
