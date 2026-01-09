package publicfiles

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	dtos "github.com/Open-Source-Life/AxolotlDrive/DTOs"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	maxChunkSize    = 10 * 1024 * 1024
	maxTotalSize    = 1 * 1024 * 1024 * 1024 * 1024
	maxSearchLength = 255
)

var allowedEditExtensions = map[string]bool{
	"txt": true, "md": true, "json": true, "yaml": true, "yml": true, "toml": true,
	"html": true, "css": true, "js": true, "ts": true, "jsx": true, "tsx": true, "xml": true,
	"csv": true, "ini": true, "env": true, "sql": true, "rs": true, "py": true, "go": true,
	"java": true, "cpp": true, "h": true, "hpp": true, "c": true,
}

type PublicFilesService struct {
	publicDir string
	wsHub     *WebSocketHub
}

func NewPublicFilesService(publicDir string, wsHub *WebSocketHub) *PublicFilesService {
	return &PublicFilesService{
		publicDir: publicDir,
		wsHub:     wsHub,
	}
}

func (p *PublicFilesService) ensurePublicDir() error {
	if _, err := os.Stat(p.publicDir); os.IsNotExist(err) {
		return os.MkdirAll(p.publicDir, 0755)
	}
	return nil
}

func (p *PublicFilesService) sanitizePathForRead(input string) (string, error) {
	decoded := input
	dangerousPatterns := []string{
		"..", "%2e%2e", "%252e%252e", "/etc/", "/root/", "/home/",
		"\\", ":", "*", "?", "\"", "<", ">", "|", "\x00", "~", "$", "&", ";", "`", "'", "(", ")", "{", "}", "[", "]",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(decoded, pattern) {
			return "", fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	normalized := strings.Trim(strings.ReplaceAll(decoded, "\\", "/"), "/")
	if normalized == "" {
		return p.publicDir, nil
	}

	clean := filepath.Join(p.publicDir, normalized)
	canonical, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("path resolution failed: %w", err)
	}

	publicCanonical, err := filepath.Abs(p.publicDir)
	if err != nil {
		return "", fmt.Errorf("public directory resolution failed: %w", err)
	}

	if !strings.HasPrefix(canonical, publicCanonical) {
		return "", fmt.Errorf("path escape attempt detected")
	}

	stat, err := os.Stat(canonical)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("directory does not exist")
		}
		return "", fmt.Errorf("path resolution failed: %w", err)
	}

	if !stat.IsDir() && !strings.HasPrefix(canonical, publicCanonical) {
		return "", fmt.Errorf("path escape attempt detected")
	}

	relPath, err := filepath.Rel(publicCanonical, canonical)
	if err != nil {
		return "", fmt.Errorf("path validation failed: %w", err)
	}

	for _, component := range strings.Split(relPath, string(filepath.Separator)) {
		if strings.HasPrefix(component, ".") && component != "." {
			return "", fmt.Errorf("access to hidden files is not allowed")
		}
		if strings.Contains(component, "\x00") || strings.Contains(component, "/") || strings.Contains(component, "\\") {
			return "", fmt.Errorf("invalid filename characters")
		}
	}

	return canonical, nil
}

func (p *PublicFilesService) sanitizePathForWrite(input string) (string, error) {
	decoded := input
	dangerousPatterns := []string{
		"..", "%2e%2e", "%252e%252e", "/etc/", "/root/", "/home/",
		"\\", ":", "*", "?", "\"", "<", ">", "|", "\x00", "~", "$", "&", ";", "`", "'", "(", ")", "{", "}", "[", "]",
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(decoded, pattern) {
			return "", fmt.Errorf("dangerous pattern detected: %s", pattern)
		}
	}

	normalized := strings.Trim(strings.ReplaceAll(decoded, "\\", "/"), "/")
	if normalized == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	clean := filepath.Join(p.publicDir, normalized)
	canonical, err := filepath.Abs(clean)
	if err != nil {
		return "", fmt.Errorf("path resolution failed: %w", err)
	}

	publicCanonical, err := filepath.Abs(p.publicDir)
	if err != nil {
		return "", fmt.Errorf("public directory resolution failed: %w", err)
	}

	if parent := filepath.Dir(canonical); parent != "" {
		parentCanonical, err := filepath.Abs(parent)
		if err != nil {
			if err := os.MkdirAll(parent, 0755); err != nil {
				return "", fmt.Errorf("parent directory resolution failed: %w", err)
			}
			parentCanonical = parent
		}

		if !strings.HasPrefix(parentCanonical, publicCanonical) {
			return "", fmt.Errorf("path escape attempt detected")
		}
	}

	fileName := filepath.Base(canonical)
	if strings.HasPrefix(fileName, ".") {
		return "", fmt.Errorf("cannot create hidden files")
	}
	if len(fileName) > 255 {
		return "", fmt.Errorf("filename too long")
	}
	if strings.Contains(fileName, "\x00") || strings.Contains(fileName, "/") || strings.Contains(fileName, "\\") {
		return "", fmt.Errorf("invalid filename characters")
	}

	return canonical, nil
}

func (p *PublicFilesService) getMimeType(filePath string) *string {
	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return nil
	}

	ext := filepath.Ext(filePath)
	if ext == "" {
		mimeType := "application/octet-stream"
		return &mimeType
	}

	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	return &mimeType
}

func (p *PublicFilesService) generateEtag(filePath string, modified *int64, size int64) string {
	if modified != nil {
		return fmt.Sprintf("\"%s-%d-%d\"", filePath, *modified, size)
	}
	return fmt.Sprintf("\"%s-%d\"", filePath, size)
}

func (p *PublicFilesService) generateUUID(data string) string {
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(data)).String()
}

func (p *PublicFilesService) ListItemsRoot(pageVal, limitVal int) (*dtos.PaginatedItems, *dtos.ErrorResponse) {
	return p.listItemsImpl(nil, pageVal, limitVal)
}

func (p *PublicFilesService) ListItems(path string, pageVal, limitVal int) (*dtos.PaginatedItems, *dtos.ErrorResponse) {
	if path == "" || path == "/" || path == "*" {
		return p.listItemsImpl(nil, pageVal, limitVal)
	}
	return p.listItemsImpl(&path, pageVal, limitVal)
}

func (p *PublicFilesService) listItemsImpl(pathPtr *string, pageVal, limitVal int) (*dtos.PaginatedItems, *dtos.ErrorResponse) {
	if err := p.ensurePublicDir(); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	base := p.publicDir
	if pathPtr != nil {
		cleanPath, err := p.sanitizePathForRead(*pathPtr)
		if err != nil {
			return nil, &dtos.ErrorResponse{
				Error:     err.Error(),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				RequestID: uuid.New().String(),
				Debug:     ptrString(err.Error()),
			}
		}
		base = cleanPath
	}

	info, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &dtos.ErrorResponse{
				Error:     "Directory not found",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				RequestID: uuid.New().String(),
				Debug:     ptrString(fmt.Sprintf("Path does not exist: %s", base)),
			}
		}
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to read directory: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if !info.IsDir() {
		return nil, &dtos.ErrorResponse{
			Error:     "Path is not a directory",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("Path is not a directory: %s", base)),
		}
	}

	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to read directory: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	var items []dtos.FileSystemItem

	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") || name == ".." || name == "." {
			continue
		}

		filePath := filepath.Join(base, name)
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(p.publicDir, filePath)
		if err != nil {
			continue
		}

		var createdAt, modifiedAt *int64
		modTime := info.ModTime().Unix()
		modifiedAt = &modTime

		items = append(items, dtos.FileSystemItem{
			ID:         p.generateUUID(relPath),
			Name:       name,
			Path:       relPath,
			Size:       info.Size(),
			IsDir:      info.IsDir(),
			CreatedAt:  createdAt,
			ModifiedAt: modifiedAt,
			MimeType:   p.getMimeType(filePath),
			Etag:       p.generateEtag(filePath, modifiedAt, info.Size()),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return strings.ToLower(items[i].Name) < strings.ToLower(items[j].Name)
	})

	page := int32(pageVal)
	if page < 1 {
		page = 1
	}
	limit := int32(limitVal)
	if limit < 10 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	total := int32(len(items))
	totalPages := (total + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if end > total {
		end = total
	}

	var paginatedItems []dtos.FileSystemItem
	if int(start) < len(items) {
		paginatedItems = items[start:end]
	}

	return &dtos.PaginatedItems{
		Items:      paginatedItems,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}, nil
}

func (p *PublicFilesService) SearchItems(query string, pageVal, limitVal int) (*dtos.PaginatedItems, *dtos.ErrorResponse) {
	queryLower := strings.ToLower(query)

	if queryLower == "" || len(queryLower) > maxSearchLength {
		return nil, &dtos.ErrorResponse{
			Error:     "Search query must be 1-255 characters",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
		}
	}

	var results []dtos.FileSystemItem

	err := filepath.Walk(p.publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		name := info.Name()
		if strings.HasPrefix(name, ".") || name == "." || name == ".." {
			return nil
		}

		if len(results) >= int((int32(pageVal) * int32(limitVal))) {
			return filepath.SkipDir
		}

		if strings.Contains(strings.ToLower(name), queryLower) {
			relPath, err := filepath.Rel(p.publicDir, path)
			if err != nil {
				return nil
			}

			var modifiedAt *int64
			modTime := info.ModTime().Unix()
			modifiedAt = &modTime

			results = append(results, dtos.FileSystemItem{
				ID:         p.generateUUID(relPath),
				Name:       name,
				Path:       relPath,
				Size:       info.Size(),
				IsDir:      info.IsDir(),
				ModifiedAt: modifiedAt,
				MimeType:   p.getMimeType(path),
				Etag:       p.generateEtag(path, modifiedAt, info.Size()),
			})
		}

		return nil
	})

	if err != nil {
		log.Debug().Err(err).Msg("Error walking directory")
	}

	page := int32(pageVal)
	if page < 1 {
		page = 1
	}
	limit := int32(limitVal)
	if limit < 10 {
		limit = 10
	}
	if limit > 500 {
		limit = 500
	}

	total := int32(len(results))
	totalPages := (total + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if end > total {
		end = total
	}

	var paginatedItems []dtos.FileSystemItem
	if int(start) < len(results) {
		paginatedItems = results[start:end]
	}

	return &dtos.PaginatedItems{
		Items:      paginatedItems,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}, nil
}

func (p *PublicFilesService) DownloadItem(path string) ([]byte, *dtos.ErrorResponse) {
	filePath, err := p.sanitizePathForRead(path)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, err := os.Stat(filePath)
	if err != nil || info.IsDir() {
		return nil, &dtos.ErrorResponse{
			Error:     "File not found",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("File does not exist or is directory: %s", filePath)),
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to read file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	return data, nil
}

func (p *PublicFilesService) DeleteItem(path string) (map[string]interface{}, *dtos.ErrorResponse) {
	target, err := p.sanitizePathForRead(path)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, err := os.Stat(target)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "File not found",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("Target does not exist: %s", target)),
		}
	}

	if info.IsDir() {
		err = os.RemoveAll(target)
	} else {
		err = os.Remove(target)
	}

	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to delete item: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	relPath, _ := filepath.Rel(p.publicDir, target)
	p.notifyWebSocket("file_deleted", map[string]interface{}{
		"path":       strings.TrimPrefix(relPath, "/"),
		"deleted_at": time.Now().Unix(),
	})

	return map[string]interface{}{
		"success": true,
		"path":    strings.TrimPrefix(relPath, "/"),
	}, nil
}

func (p *PublicFilesService) EditFile(filePath, content string) (map[string]interface{}, *dtos.ErrorResponse) {
	file, err := p.sanitizePathForWrite(filePath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if _, err := os.Stat(file); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "File not found",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("File does not exist: %s", file)),
		}
	}

	ext := strings.TrimPrefix(filepath.Ext(file), ".")
	if !allowedEditExtensions[ext] {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("File type not editable: .%s", ext),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("Unsupported extension: %s", ext)),
		}
	}

	if len(content) > 10*1024*1024 {
		return nil, &dtos.ErrorResponse{
			Error:     "File content too large (maximum 10MB)",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("Content size: %d bytes", len(content))),
		}
	}

	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to write file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	newInfo, _ := os.Stat(file)
	modTime := newInfo.ModTime().Unix()

	relPath, _ := filepath.Rel(p.publicDir, file)
	p.notifyWebSocket("file_updated", map[string]interface{}{
		"path":        strings.TrimPrefix(relPath, "/"),
		"size":        newInfo.Size(),
		"modified_at": modTime,
		"etag":        p.generateEtag(file, &modTime, newInfo.Size()),
	})

	return map[string]interface{}{
		"success":     true,
		"path":        strings.TrimPrefix(relPath, "/"),
		"size":        newInfo.Size(),
		"modified_at": modTime,
		"etag":        p.generateEtag(file, &modTime, newInfo.Size()),
	}, nil
}

func (p *PublicFilesService) UploadFile(filePath string, data io.Reader) (map[string]interface{}, *dtos.ErrorResponse) {
	file, err := p.sanitizePathForWrite(filePath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create parent directories: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	f, err := os.Create(file)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}
	defer f.Close()

	var totalBytes int64
	buf := make([]byte, maxChunkSize)
	uploadID := uuid.New().String()

	for {
		n, err := data.Read(buf)
		if n > 0 {
			totalBytes += int64(n)

			if totalBytes > maxTotalSize {
				os.Remove(file)
				return nil, &dtos.ErrorResponse{
					Error:     fmt.Sprintf("File size exceeds maximum limit (%.2f GB)", float64(maxTotalSize)/1024/1024/1024),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					RequestID: uuid.New().String(),
					Debug:     ptrString(fmt.Sprintf("Total bytes: %d", totalBytes)),
				}
			}

			if _, err := f.Write(buf[:n]); err != nil {
				os.Remove(file)
				return nil, &dtos.ErrorResponse{
					Error:     fmt.Sprintf("Failed to write chunk: %v", err),
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					RequestID: uuid.New().String(),
					Debug:     ptrString(err.Error()),
				}
			}
		}

		if err != nil && err != io.EOF {
			os.Remove(file)
			return nil, &dtos.ErrorResponse{
				Error:     fmt.Sprintf("Failed to read chunk: %v", err),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				RequestID: uuid.New().String(),
				Debug:     ptrString(err.Error()),
			}
		}

		if err == io.EOF {
			break
		}
	}

	os.Chmod(file, 0644)

	info, _ := os.Stat(file)
	modTime := info.ModTime().Unix()
	relPath, _ := filepath.Rel(p.publicDir, file)

	p.notifyWebSocket("file_created", map[string]interface{}{
		"path":        strings.TrimPrefix(relPath, "/"),
		"size":        totalBytes,
		"mime_type":   p.getMimeType(file),
		"modified_at": modTime,
		"etag":        p.generateEtag(file, &modTime, totalBytes),
	})

	return map[string]interface{}{
		"success":     true,
		"path":        strings.TrimPrefix(relPath, "/"),
		"size_bytes":  totalBytes,
		"mime_type":   p.getMimeType(file),
		"modified_at": modTime,
		"etag":        p.generateEtag(file, &modTime, totalBytes),
		"upload_id":   uploadID,
	}, nil
}

func (p *PublicFilesService) CreateFolder(path string) (map[string]interface{}, *dtos.ErrorResponse) {
	dirPath, err := p.sanitizePathForWrite(path)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.MkdirAll(filepath.Dir(dirPath), 0755); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create parent directories: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	err = os.Mkdir(dirPath, 0755)
	if err != nil && !os.IsExist(err) {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create folder: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if os.IsExist(err) {
		return nil, &dtos.ErrorResponse{
			Error:     "Directory already exists",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	os.Chmod(dirPath, 0755)
	relPath, _ := filepath.Rel(p.publicDir, dirPath)
	createdAt := time.Now().Unix()

	p.notifyWebSocket("folder_created", map[string]interface{}{
		"path":       strings.TrimPrefix(relPath, "/"),
		"created_at": createdAt,
	})

	return map[string]interface{}{
		"success":    true,
		"path":       strings.TrimPrefix(relPath, "/"),
		"type":       "directory",
		"created_at": createdAt,
	}, nil
}

func (p *PublicFilesService) CreateFile(path string) (map[string]interface{}, *dtos.ErrorResponse) {
	filePath, err := p.sanitizePathForWrite(path)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create parent directories: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	relPath, _ := filepath.Rel(p.publicDir, filePath)
	createdAt := time.Now().Unix()

	p.notifyWebSocket("file_created", map[string]interface{}{
		"path":       strings.TrimPrefix(relPath, "/"),
		"size":       0,
		"type":       "file",
		"created_at": createdAt,
		"etag":       p.generateEtag(filePath, nil, 0),
	})

	return map[string]interface{}{
		"success":    true,
		"path":       strings.TrimPrefix(relPath, "/"),
		"type":       "file",
		"size_bytes": 0,
		"created_at": createdAt,
		"etag":       p.generateEtag(filePath, nil, 0),
	}, nil
}

func (p *PublicFilesService) RenameFile(oldPath, newPath string) (map[string]interface{}, *dtos.ErrorResponse) {
	oldPathSanitized, err := p.sanitizePathForWrite(oldPath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	newPathSanitized, err := p.sanitizePathForWrite(newPath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if _, err := os.Stat(oldPathSanitized); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Source file does not exist",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Source file does not exist"),
		}
	}

	if _, err := os.Stat(newPathSanitized); err == nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Destination file already exists",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Destination file already exists"),
		}
	}

	if err := os.Rename(oldPathSanitized, newPathSanitized); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to rename file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	oldRel, _ := filepath.Rel(p.publicDir, oldPathSanitized)
	newRel, _ := filepath.Rel(p.publicDir, newPathSanitized)

	p.notifyWebSocket("file_renamed", map[string]interface{}{
		"old_path":  strings.TrimPrefix(oldRel, "/"),
		"new_path":  strings.TrimPrefix(newRel, "/"),
		"timestamp": time.Now().Unix(),
	})

	return map[string]interface{}{
		"success":  true,
		"message":  "File renamed successfully",
		"old_path": strings.TrimPrefix(oldRel, "/"),
		"new_path": strings.TrimPrefix(newRel, "/"),
	}, nil
}

func (p *PublicFilesService) MoveFile(source, destination string) (map[string]interface{}, *dtos.ErrorResponse) {
	sourcePath, err := p.sanitizePathForWrite(source)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	destPath, err := p.sanitizePathForWrite(destination)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if _, err := os.Stat(sourcePath); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Source file does not exist",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Source file does not exist"),
		}
	}

	if _, err := os.Stat(destPath); err == nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Destination file already exists",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Destination file already exists"),
		}
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)

	if err := os.Rename(sourcePath, destPath); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to move file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, _ := os.Stat(destPath)
	modTime := info.ModTime().Unix()

	sourceRel, _ := filepath.Rel(p.publicDir, sourcePath)
	destRel, _ := filepath.Rel(p.publicDir, destPath)

	p.notifyWebSocket("file_moved", map[string]interface{}{
		"source_path":      strings.TrimPrefix(sourceRel, "/"),
		"destination_path": strings.TrimPrefix(destRel, "/"),
		"size":             info.Size(),
		"modified_at":      modTime,
		"timestamp":        time.Now().Unix(),
	})

	return map[string]interface{}{
		"success":     true,
		"message":     "File moved successfully",
		"source":      strings.TrimPrefix(sourceRel, "/"),
		"destination": strings.TrimPrefix(destRel, "/"),
		"size":        info.Size(),
		"modified_at": modTime,
	}, nil
}

func (p *PublicFilesService) CopyFile(source, destination string) (map[string]interface{}, *dtos.ErrorResponse) {
	sourcePath, err := p.sanitizePathForWrite(source)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	destPath, err := p.sanitizePathForWrite(destination)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if _, err := os.Stat(sourcePath); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Source file does not exist",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Source file does not exist"),
		}
	}

	if _, err := os.Stat(destPath); err == nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Destination file already exists",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Destination file already exists"),
		}
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)

	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to copy file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to copy file: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, _ := os.Stat(destPath)
	modTime := info.ModTime().Unix()

	sourceRel, _ := filepath.Rel(p.publicDir, sourcePath)
	destRel, _ := filepath.Rel(p.publicDir, destPath)

	p.notifyWebSocket("file_copied", map[string]interface{}{
		"source_path":      strings.TrimPrefix(sourceRel, "/"),
		"destination_path": strings.TrimPrefix(destRel, "/"),
		"size":             info.Size(),
		"modified_at":      modTime,
		"timestamp":        time.Now().Unix(),
	})

	return map[string]interface{}{
		"success":      true,
		"message":      "File copied successfully",
		"source":       strings.TrimPrefix(sourceRel, "/"),
		"destination":  strings.TrimPrefix(destRel, "/"),
		"size":         info.Size(),
		"bytes_copied": int64(len(data)),
		"modified_at":  modTime,
	}, nil

}

func (p *PublicFilesService) UploadFolder(folderPath string, files map[string][]byte) (map[string]interface{}, *dtos.ErrorResponse) {
	folderPathSanitized, err := p.sanitizePathForWrite(folderPath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if err := os.MkdirAll(folderPathSanitized, 0755); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to create folder: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	uploadedCount := 0
	for fileName, fileData := range files {
		filePath := filepath.Join(folderPathSanitized, fileName)

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			continue
		}

		if err := os.WriteFile(filePath, fileData, 0644); err != nil {
			continue
		}
		uploadedCount++
	}

	relPath, _ := filepath.Rel(p.publicDir, folderPathSanitized)
	createdAt := time.Now().Unix()

	p.notifyWebSocket("folder_uploaded", map[string]interface{}{
		"path":       strings.TrimPrefix(relPath, "/"),
		"files_count": uploadedCount,
		"created_at": createdAt,
	})

	return map[string]interface{}{
		"success":     true,
		"path":        strings.TrimPrefix(relPath, "/"),
		"type":        "directory",
		"files_count": uploadedCount,
		"created_at":  createdAt,
	}, nil
}

func (p *PublicFilesService) DownloadFolder(folderPath string) (map[string][]byte, *dtos.ErrorResponse) {
	folderPathSanitized, err := p.sanitizePathForRead(folderPath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, err := os.Stat(folderPathSanitized)
	if err != nil || !info.IsDir() {
		return nil, &dtos.ErrorResponse{
			Error:     "Folder not found or is not a directory",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(fmt.Sprintf("Folder does not exist or is not directory: %s", folderPathSanitized)),
		}
	}

	files := make(map[string][]byte)

	err = filepath.Walk(folderPathSanitized, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			relPath, err := filepath.Rel(folderPathSanitized, path)
			if err != nil {
				return nil
			}

			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			files[relPath] = data
		}
		return nil
	})

	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to read folder: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	return files, nil
}

func (p *PublicFilesService) RenameFolder(oldPath, newPath string) (map[string]interface{}, *dtos.ErrorResponse) {
	return p.RenameFile(oldPath, newPath)
}

func (p *PublicFilesService) MoveFolder(source, destination string) (map[string]interface{}, *dtos.ErrorResponse) {
	return p.MoveFile(source, destination)
}

func (p *PublicFilesService) CopyFolder(source, destination string) (map[string]interface{}, *dtos.ErrorResponse) {
	sourcePath, err := p.sanitizePathForWrite(source)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	destPath, err := p.sanitizePathForWrite(destination)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     err.Error(),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	if _, err := os.Stat(sourcePath); err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Source folder does not exist",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Source folder does not exist"),
		}
	}

	if _, err := os.Stat(destPath); err == nil {
		return nil, &dtos.ErrorResponse{
			Error:     "Destination folder already exists",
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString("Destination folder already exists"),
		}
	}

	os.MkdirAll(filepath.Dir(destPath), 0755)

	err = p.copyDirectory(sourcePath, destPath)
	if err != nil {
		return nil, &dtos.ErrorResponse{
			Error:     fmt.Sprintf("Failed to copy folder: %v", err),
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			RequestID: uuid.New().String(),
			Debug:     ptrString(err.Error()),
		}
	}

	info, _ := os.Stat(destPath)
	modTime := info.ModTime().Unix()

	sourceRel, _ := filepath.Rel(p.publicDir, sourcePath)
	destRel, _ := filepath.Rel(p.publicDir, destPath)

	p.notifyWebSocket("folder_copied", map[string]interface{}{
		"source_path":      strings.TrimPrefix(sourceRel, "/"),
		"destination_path": strings.TrimPrefix(destPath, "/"),
		"size":             info.Size(),
		"modified_at":      modTime,
		"timestamp":        time.Now().Unix(),
	})

	return map[string]interface{}{
		"success":      true,
		"message":      "Folder copied successfully",
		"source":       strings.TrimPrefix(sourceRel, "/"),
		"destination":  strings.TrimPrefix(destRel, "/"),
		"size":         info.Size(),
		"modified_at":  modTime,
	}, nil
}

func (p *PublicFilesService) copyDirectory(src, dst string) error {
    entries, err := os.ReadDir(src)
    if err != nil {
        return err
    }

    if err := os.MkdirAll(dst, 0755); err != nil {
        return err
    }

    for _, entry := range entries {
        srcPath := filepath.Join(src, entry.Name())
        dstPath := filepath.Join(dst, entry.Name())

        if entry.IsDir() {
            if err := p.copyDirectory(srcPath, dstPath); err != nil {
                return err
            }
        } else {
            data, err := os.ReadFile(srcPath)
            if err != nil {
                return err
            }
            if err := os.WriteFile(dstPath, data, 0644); err != nil {
                return err
            }
        }
    }

    return nil
}

func (p *PublicFilesService) notifyWebSocket(eventType string, data interface{}) {
	if p.wsHub != nil {
		msg := dtos.WebSocketMessage{
			EventType: eventType,
			Data:      data,
			Timestamp: time.Now().Unix(),
		}
		p.wsHub.Broadcast(msg)
	}
}

func ptrString(s string) *string {
	return &s
}
