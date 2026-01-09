package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Open-Source-Life/AxolotlDrive/middlewares"
	publicfiles "github.com/Open-Source-Life/AxolotlDrive/services/public_files"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func setupTestApp(t *testing.T) (*fiber.App, string) {
	tmpDir := t.TempDir()
	publicDir := filepath.Join(tmpDir, "public")

	// Create the public directory that the service expects
	err := os.MkdirAll(publicDir, 0755)
	assert.NoError(t, err)

	app := fiber.New()
	app.Use(middlewares.Recovery())
	app.Use(middlewares.Logger())
	app.Use(middlewares.CORS())

	// Create the WebSocket hub
	wsHub := publicfiles.NewWebSocketHub()
	go wsHub.Run()

	// Create the public files service with the test public directory
	publicFilesService := publicfiles.NewPublicFilesService(publicDir, wsHub)

	v1 := app.Group("/api/v1")
	setupTestRoutes(&v1, publicFilesService)

	return app, publicDir
}

// Custom route setup for testing
func setupTestRoutes(app *fiber.Router, publicFilesService *publicfiles.PublicFilesService) {
	(*app).Get("/healthz", func(c *fiber.Ctx) error {
		return testHealthCheck(c)
	})

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

	(*app).Get("/files/download/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		data, errResp := publicFilesService.DownloadItem(path)
		if errResp != nil {
			return c.Status(fiber.StatusNotFound).JSON(errResp)
		}
		return c.Send(data)
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

	(*app).Get("/files/download-folder/*", func(c *fiber.Ctx) error {
		path := strings.TrimPrefix(c.Params("*"), "/")
		files, errResp := publicFilesService.DownloadFolder(path)
		if errResp != nil {
			return c.Status(fiber.StatusNotFound).JSON(errResp)
		}
		return c.JSON(files)
	})
}

// testHealthCheck is a test version of the health check
func testHealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": "test",
	})
}

func TestIntegration_HealthCheck(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("GET", "/api/v1/healthz", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), "healthy")
}

func TestIntegration_ListFilesEmpty(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("GET", "/api/v1/files", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result)
}

func TestIntegration_CreateAndListFiles(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	os.WriteFile(filepath.Join(tmpDir, "test.txt"), []byte("test"), 0644)

	req, _ := http.NewRequest("GET", "/api/v1/files", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_CreateFile(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("POST", "/api/v1/files/create-file/newfile.txt", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_CreateFolder(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("POST", "/api/v1/files/mkdir/newfolder", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_DeleteFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	os.MkdirAll(filepath.Dir(testFile), 0755)
	os.WriteFile(testFile, []byte("test"), 0644)

	req, _ := http.NewRequest("DELETE", "/api/v1/files/test.txt", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_RenameFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	oldFile := filepath.Join(tmpDir, "old.txt")
	os.MkdirAll(filepath.Dir(oldFile), 0755)
	os.WriteFile(oldFile, []byte("test"), 0644)

	body := bytes.NewBufferString(`{"old_path":"old.txt","new_path":"new.txt"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/rename", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_MoveFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	os.Mkdir(filepath.Join(tmpDir, "folder"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	body := bytes.NewBufferString(`{"source":"file.txt","destination":"folder/file.txt"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/move", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_CopyFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	os.WriteFile(filepath.Join(tmpDir, "file.txt"), []byte("test"), 0644)

	body := bytes.NewBufferString(`{"source":"file.txt","destination":"copy.txt"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/copy", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_EditFile(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	testFile := filepath.Join(tmpDir, "file.txt")
	os.MkdirAll(filepath.Dir(testFile), 0755)
	os.WriteFile(testFile, []byte("old"), 0644)

	req, _ := http.NewRequest("PUT", "/api/v1/files/edit/file.txt", bytes.NewBufferString("new content"))
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_SearchFiles(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	os.WriteFile(filepath.Join(tmpDir, "document.txt"), []byte("test"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "image.png"), []byte("test"), 0644)

	req, _ := http.NewRequest("GET", "/api/v1/files/search?q=document", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result)
}

func TestIntegration_UploadFile(t *testing.T) {
	app, _ := setupTestApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, _ := writer.CreateFormFile("file", "test.txt")
	file.Write([]byte("test content"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/files/upload/test.txt", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_PathTraversalAttack(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("GET", "/api/v1/files/../../../../etc/passwd", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_CreateNestedStructure(t *testing.T) {
	app, _ := setupTestApp(t)

	req1, _ := http.NewRequest("POST", "/api/v1/files/mkdir/folder1", nil)
	resp1, _ := app.Test(req1, -1)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	req2, _ := http.NewRequest("POST", "/api/v1/files/mkdir/folder1/folder2", nil)
	resp2, _ := app.Test(req2, -1)
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	req3, _ := http.NewRequest("POST", "/api/v1/files/create-file/folder1/folder2/file.txt", nil)
	resp3, _ := app.Test(req3, -1)
	assert.Equal(t, http.StatusOK, resp3.StatusCode)
}

// New tests for the additional functionality
func TestIntegration_UploadFolder(t *testing.T) {
	app, _ := setupTestApp(t)

	// Create JSON payload for upload-folder
	payload := `{
		"file1.txt": "SGVsbG8gV29ybGQh",
		"file2.txt": "VGhpcyBpcyBhIHRlc3Q="
	}` // Base64 encoded content

	req, _ := http.NewRequest("POST", "/api/v1/files/upload-folder/test_folder", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
	assert.Equal(t, "directory", result["type"])
}

func TestIntegration_DownloadFolder(t *testing.T) {
	app, _ := setupTestApp(t)

	// Create a folder with some files first using the service directly
	publicDir := t.TempDir()

	// Create the folder structure in the test
	folderPath := filepath.Join(publicDir, "test_folder")
	os.MkdirAll(folderPath, 0755)
	os.WriteFile(filepath.Join(folderPath, "file1.txt"), []byte("Hello World!"), 0644)
	os.WriteFile(filepath.Join(folderPath, "file2.txt"), []byte("This is a test"), 0644)

	// Now test download-folder via the API route
	req, _ := http.NewRequest("GET", "/api/v1/files/download-folder/test_folder", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.NotNil(t, result)
	assert.Contains(t, result, "file1.txt")
	assert.Contains(t, result, "file2.txt")
}

func TestIntegration_RenameFolder(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	// Create a folder first
	folderPath := filepath.Join(tmpDir, "old_folder")
	os.Mkdir(folderPath, 0755)

	body := bytes.NewBufferString(`{"old_path":"old_folder","new_path":"new_folder"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/rename-folder", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_MoveFolder(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	// Create source and destination folders
	sourcePath := filepath.Join(tmpDir, "source_folder")
	destPath := filepath.Join(tmpDir, "dest_folder")
	os.Mkdir(sourcePath, 0755)
	os.Mkdir(destPath, 0755)
	os.WriteFile(filepath.Join(sourcePath, "file.txt"), []byte("test"), 0644)

	body := bytes.NewBufferString(`{"source":"source_folder","destination":"moved_folder"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/move-folder", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_CopyFolder(t *testing.T) {
	app, tmpDir := setupTestApp(t)

	// Create source folder with a file
	sourcePath := filepath.Join(tmpDir, "source_folder")
	os.Mkdir(sourcePath, 0755)
	os.WriteFile(filepath.Join(sourcePath, "file.txt"), []byte("test"), 0644)

	body := bytes.NewBufferString(`{"source":"source_folder","destination":"copied_folder"}`)
	req, _ := http.NewRequest("POST", "/api/v1/files/copy-folder", body)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	assert.Equal(t, true, result["success"])
}

func TestIntegration_UploadFileToNestedPath(t *testing.T) {
	app, _ := setupTestApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, _ := writer.CreateFormFile("file", "nested_file.txt")
	file.Write([]byte("nested content"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/files/upload/nested/path/nested_file.txt", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestIntegration_DownloadFileFromNestedPath(t *testing.T) {
	app, _ := setupTestApp(t)

	// First upload a file to create the nested structure
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, _ := writer.CreateFormFile("file", "nested_file.txt")
	file.Write([]byte("nested content"))
	writer.Close()

	// Upload the file to nested path first
	req1, _ := http.NewRequest("POST", "/api/v1/files/upload/nested/path/nested_file.txt", body)
	req1.Header.Set("Content-Type", writer.FormDataContentType())
	resp1, _ := app.Test(req1, -1)
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	// Now test downloading the file
	req2, _ := http.NewRequest("GET", "/api/v1/files/download/nested/path/nested_file.txt", nil)
	resp2, _ := app.Test(req2, -1)

	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

// Security tests
func TestIntegration_PathTraversalAttackNested(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("GET", "/api/v1/files/nested/../../../etc/passwd", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_PathTraversalAttackUpload(t *testing.T) {
	app, _ := setupTestApp(t)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, _ := writer.CreateFormFile("file", "test.txt")
	file.Write([]byte("test content"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/files/upload/../../../etc/passwd", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, _ := app.Test(req, -1)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegration_PathTraversalAttackMkdir(t *testing.T) {
	app, _ := setupTestApp(t)

	req, _ := http.NewRequest("POST", "/api/v1/files/mkdir/../../../etc", nil)
	resp, _ := app.Test(req, -1)

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

