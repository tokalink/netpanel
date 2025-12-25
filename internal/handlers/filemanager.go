package handlers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

// FileInfo represents file/folder information
type FileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"is_dir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	Ext     string    `json:"ext"`
}

// GetBaseDir returns the base directory for file manager
func getFileManagerBaseDir() string {
	cwd, _ := os.Getwd()
	return filepath.Join(cwd, "server")
}

// ListFiles lists files in a directory
func ListFiles(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()
	requestPath := c.Query("path", "/")

	// Sanitize path
	cleanPath := filepath.Clean(requestPath)
	if cleanPath == "." {
		cleanPath = ""
	}

	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check - ensure path is within base dir
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		relPath := filepath.Join(cleanPath, entry.Name())
		ext := ""
		if !entry.IsDir() {
			ext = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Path:    "/" + strings.ReplaceAll(relPath, "\\", "/"),
			IsDir:   entry.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Ext:     ext,
		})
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return c.JSON(fiber.Map{
		"path":  "/" + strings.ReplaceAll(cleanPath, "\\", "/"),
		"files": files,
	})
}

// ReadFile reads file content
func ReadFileContent(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()
	requestPath := c.Query("path", "")

	if requestPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is required",
		})
	}

	cleanPath := filepath.Clean(requestPath)
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Cannot read directory",
		})
	}

	// Limit file size for reading
	if info.Size() > 5*1024*1024 { // 5MB limit
		return c.Status(400).JSON(fiber.Map{
			"error": "File too large to read",
		})
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"path":    requestPath,
		"content": string(content),
		"size":    info.Size(),
	})
}

// SaveFile saves file content
func SaveFileContent(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()

	type SaveRequest struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}

	var req SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	cleanPath := filepath.Clean(req.Path)
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Ensure parent directory exists
	os.MkdirAll(filepath.Dir(fullPath), 0755)

	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File saved",
	})
}

// CreateFolder creates a new folder
func CreateFolder(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()

	type CreateRequest struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	cleanPath := filepath.Clean(filepath.Join(req.Path, req.Name))
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Folder created",
	})
}

// CreateFile creates a new file
func CreateFile(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()

	type CreateRequest struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}

	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	cleanPath := filepath.Clean(filepath.Join(req.Path, req.Name))
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	// Check if file exists
	if _, err := os.Stat(fullPath); err == nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "File already exists",
		})
	}

	// Create empty file
	file, err := os.Create(fullPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	file.Close()

	return c.JSON(fiber.Map{
		"success": true,
		"message": "File created",
	})
}

// DeleteItem deletes a file or folder
func DeleteItem(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()
	requestPath := c.Query("path", "")

	if requestPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is required",
		})
	}

	cleanPath := filepath.Clean(requestPath)
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) || fullPath == baseDir {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	if err := os.RemoveAll(fullPath); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Deleted successfully",
	})
}

// RenameItem renames a file or folder
func RenameItem(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()

	type RenameRequest struct {
		OldPath string `json:"old_path"`
		NewName string `json:"new_name"`
	}

	var req RenameRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request",
		})
	}

	oldFullPath := filepath.Join(baseDir, filepath.Clean(req.OldPath))
	newFullPath := filepath.Join(filepath.Dir(oldFullPath), req.NewName)

	// Security check
	if !strings.HasPrefix(oldFullPath, baseDir) || !strings.HasPrefix(newFullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	if err := os.Rename(oldFullPath, newFullPath); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Renamed successfully",
	})
}

// UploadFile handles file upload
func UploadFile(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()
	uploadPath := c.FormValue("path", "/")

	cleanPath := filepath.Clean(uploadPath)
	targetDir := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(targetDir, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "No file uploaded",
		})
	}

	// Ensure directory exists
	os.MkdirAll(targetDir, 0755)

	targetPath := filepath.Join(targetDir, file.Filename)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer src.Close()

	// Create destination file
	dst, err := os.Create(targetPath)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	defer dst.Close()

	// Copy content
	if _, err := io.Copy(dst, src); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":  true,
		"message":  "File uploaded",
		"filename": file.Filename,
	})
}

// DownloadFile handles file download
func DownloadFile(c *fiber.Ctx) error {
	baseDir := getFileManagerBaseDir()
	requestPath := c.Query("path", "")

	if requestPath == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Path is required",
		})
	}

	cleanPath := filepath.Clean(requestPath)
	fullPath := filepath.Join(baseDir, cleanPath)

	// Security check
	if !strings.HasPrefix(fullPath, baseDir) {
		return c.Status(403).JSON(fiber.Map{
			"error": "Access denied",
		})
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "File not found",
		})
	}

	if info.IsDir() {
		return c.Status(400).JSON(fiber.Map{
			"error": "Cannot download directory",
		})
	}

	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filepath.Base(fullPath)))
	return c.SendFile(fullPath)
}
