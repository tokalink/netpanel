package appstore

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"vps-panel/internal/database"
	"vps-panel/internal/models"
)

// PortablePackage defines a downloadable portable package
type PortablePackage struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Versions    []PortableVersion `json:"versions"`
	InstallPath string            `json:"install_path"` // relative path like "database/mysql"
	Executable  map[string]string `json:"executable"`   // OS -> executable name
	ConfigFile  string            `json:"config_file,omitempty"`
	Ports       []int             `json:"ports,omitempty"`
}

type PortableVersion struct {
	Version   string            `json:"version"`
	Latest    bool              `json:"latest,omitempty"`
	LTS       bool              `json:"lts,omitempty"`
	Downloads map[string]string `json:"downloads"` // OS/arch -> download URL
}

// GetBaseDir returns the base directory for portable installations
func GetBaseDir() string {
	execPath, err := os.Executable()
	if err != nil {
		return "./server"
	}
	return filepath.Join(filepath.Dir(execPath), "server")
}

// PortableCatalog contains all portable packages
var PortableCatalog = []PortablePackage{
	{
		ID:          "mysql",
		Name:        "MySQL Server",
		Description: "Open-source relational database",
		Category:    "database",
		InstallPath: "database/mysql",
		Executable:  map[string]string{"windows": "bin/mysqld.exe", "linux": "bin/mysqld", "darwin": "bin/mysqld"},
		ConfigFile:  "my.cnf",
		Ports:       []int{3306},
		Versions: []PortableVersion{
			{
				Version: "8.0.35",
				Latest:  true,
				Downloads: map[string]string{
					"windows/amd64": "https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.35-winx64.zip",
					"linux/amd64":   "https://dev.mysql.com/get/Downloads/MySQL-8.0/mysql-8.0.35-linux-glibc2.17-x86_64.tar.xz",
				},
			},
			{
				Version: "5.7.44",
				Downloads: map[string]string{
					"windows/amd64": "https://dev.mysql.com/get/Downloads/MySQL-5.7/mysql-5.7.44-winx64.zip",
					"linux/amd64":   "https://dev.mysql.com/get/Downloads/MySQL-5.7/mysql-5.7.44-linux-glibc2.12-x86_64.tar.gz",
				},
			},
		},
	},
	{
		ID:          "mariadb",
		Name:        "MariaDB",
		Description: "Community-developed MySQL fork",
		Category:    "database",
		InstallPath: "database/mariadb",
		Executable:  map[string]string{"windows": "bin/mariadbd.exe", "linux": "bin/mariadbd", "darwin": "bin/mariadbd"},
		Ports:       []int{3306},
		Versions: []PortableVersion{
			{
				Version: "11.2.2",
				Latest:  true,
				Downloads: map[string]string{
					"windows/amd64": "https://archive.mariadb.org/mariadb-11.2.2/winx64-packages/mariadb-11.2.2-winx64.zip",
					"linux/amd64":   "https://archive.mariadb.org/mariadb-11.2.2/bintar-linux-systemd-x86_64/mariadb-11.2.2-linux-systemd-x86_64.tar.gz",
				},
			},
			{
				Version: "10.11.6",
				LTS:     true,
				Downloads: map[string]string{
					"windows/amd64": "https://archive.mariadb.org/mariadb-10.11.6/winx64-packages/mariadb-10.11.6-winx64.zip",
					"linux/amd64":   "https://archive.mariadb.org/mariadb-10.11.6/bintar-linux-systemd-x86_64/mariadb-10.11.6-linux-systemd-x86_64.tar.gz",
				},
			},
		},
	},
	{
		ID:          "redis",
		Name:        "Redis",
		Description: "In-memory data structure store",
		Category:    "database",
		InstallPath: "database/redis",
		Executable:  map[string]string{"windows": "redis-server.exe", "linux": "src/redis-server", "darwin": "src/redis-server"},
		Ports:       []int{6379},
		Versions: []PortableVersion{
			{
				Version: "7.2.3",
				Latest:  true,
				Downloads: map[string]string{
					"windows/amd64": "https://github.com/tporadowski/redis/releases/download/v7.2.3/Redis-7.2.3-Windows-x64.zip",
					"linux/amd64":   "https://download.redis.io/releases/redis-7.2.3.tar.gz",
				},
			},
		},
	},
	{
		ID:          "php",
		Name:        "PHP",
		Description: "Server-side scripting language",
		Category:    "runtime",
		InstallPath: "runtime/php",
		Executable:  map[string]string{"windows": "php.exe", "linux": "bin/php", "darwin": "bin/php"},
		Versions: []PortableVersion{
			{
				Version: "8.4.16",
				Latest:  true,
				Downloads: map[string]string{
					"windows/amd64": "https://windows.php.net/downloads/releases/php-8.4.16-nts-Win32-vs17-x64.zip",
				},
			},
			{
				Version: "8.3.29",
				Downloads: map[string]string{
					"windows/amd64": "https://windows.php.net/downloads/releases/php-8.3.29-nts-Win32-vs16-x64.zip",
				},
			},
			{
				Version: "8.2.30",
				Downloads: map[string]string{
					"windows/amd64": "https://windows.php.net/downloads/releases/php-8.2.30-nts-Win32-vs16-x64.zip",
				},
			},
			{
				Version: "8.1.34",
				Downloads: map[string]string{
					"windows/amd64": "https://windows.php.net/downloads/releases/php-8.1.34-nts-Win32-vs16-x64.zip",
				},
			},
		},
	},
	{
		ID:          "nodejs",
		Name:        "Node.js",
		Description: "JavaScript runtime",
		Category:    "runtime",
		InstallPath: "runtime/nodejs",
		Executable:  map[string]string{"windows": "node.exe", "linux": "bin/node", "darwin": "bin/node"},
		Versions: []PortableVersion{
			{
				Version: "20.10.0",
				Latest:  true,
				LTS:     true,
				Downloads: map[string]string{
					"windows/amd64": "https://nodejs.org/dist/v20.10.0/node-v20.10.0-win-x64.zip",
					"linux/amd64":   "https://nodejs.org/dist/v20.10.0/node-v20.10.0-linux-x64.tar.xz",
					"darwin/amd64":  "https://nodejs.org/dist/v20.10.0/node-v20.10.0-darwin-x64.tar.gz",
					"darwin/arm64":  "https://nodejs.org/dist/v20.10.0/node-v20.10.0-darwin-arm64.tar.gz",
				},
			},
			{
				Version: "18.19.0",
				LTS:     true,
				Downloads: map[string]string{
					"windows/amd64": "https://nodejs.org/dist/v18.19.0/node-v18.19.0-win-x64.zip",
					"linux/amd64":   "https://nodejs.org/dist/v18.19.0/node-v18.19.0-linux-x64.tar.xz",
					"darwin/amd64":  "https://nodejs.org/dist/v18.19.0/node-v18.19.0-darwin-x64.tar.gz",
				},
			},
		},
	},
	{
		ID:          "nginx",
		Name:        "Nginx",
		Description: "High-performance web server",
		Category:    "webserver",
		InstallPath: "webserver/nginx",
		Executable:  map[string]string{"windows": "nginx.exe", "linux": "sbin/nginx", "darwin": "sbin/nginx"},
		ConfigFile:  "conf/nginx.conf",
		Ports:       []int{80, 443},
		Versions: []PortableVersion{
			{
				Version: "1.25.3",
				Latest:  true,
				Downloads: map[string]string{
					"windows/amd64": "https://nginx.org/download/nginx-1.25.3.zip",
					"linux/amd64":   "https://nginx.org/download/nginx-1.25.3.tar.gz",
				},
			},
			{
				Version: "1.24.0",
				Downloads: map[string]string{
					"windows/amd64": "https://nginx.org/download/nginx-1.24.0.zip",
					"linux/amd64":   "https://nginx.org/download/nginx-1.24.0.tar.gz",
				},
			},
		},
	},
	{
		ID:          "phpmyadmin",
		Name:        "phpMyAdmin",
		Description: "MySQL web administration tool",
		Category:    "tools",
		InstallPath: "addons/phpmyadmin",
		Versions: []PortableVersion{
			{
				Version: "5.2.1",
				Latest:  true,
				Downloads: map[string]string{
					"all": "https://files.phpmyadmin.net/phpMyAdmin/5.2.1/phpMyAdmin-5.2.1-all-languages.zip",
				},
			},
		},
	},
	{
		ID:          "adminer",
		Name:        "Adminer",
		Description: "Lightweight database management",
		Category:    "tools",
		InstallPath: "addons/adminer",
		Versions: []PortableVersion{
			{
				Version: "4.8.1",
				Latest:  true,
				Downloads: map[string]string{
					"all": "https://github.com/vrana/adminer/releases/download/v4.8.1/adminer-4.8.1.php",
				},
			},
		},
	},
	{
		ID:          "composer",
		Name:        "Composer",
		Description: "PHP dependency manager",
		Category:    "tools",
		InstallPath: "addons/composer",
		Executable:  map[string]string{"windows": "composer.phar", "linux": "composer.phar", "darwin": "composer.phar"},
		Versions: []PortableVersion{
			{
				Version: "2.6.6",
				Latest:  true,
				Downloads: map[string]string{
					"all": "https://getcomposer.org/download/2.6.6/composer.phar",
				},
			},
		},
	},
}

// GetPortablePackages returns all portable packages
func GetPortablePackages() []PortablePackage {
	// Add installed status to each
	for i := range PortableCatalog {
		PortableCatalog[i] = checkInstalledVersions(PortableCatalog[i])
	}
	return PortableCatalog
}

// GetPortablePackageByID returns a package by ID
func GetPortablePackageByID(id string) *PortablePackage {
	for _, pkg := range PortableCatalog {
		if pkg.ID == id {
			return &pkg
		}
	}
	return nil
}

// Check which versions are installed
func checkInstalledVersions(pkg PortablePackage) PortablePackage {
	baseDir := GetBaseDir()
	for _, ver := range pkg.Versions {
		versionPath := filepath.Join(baseDir, pkg.InstallPath, ver.Version)
		if _, err := os.Stat(versionPath); err == nil {
			// version is installed
		}
	}
	return pkg
}

// GetDownloadURL returns the download URL for current OS/arch
func GetDownloadURL(pkg *PortablePackage, version string) (string, error) {
	var targetVersion *PortableVersion
	for _, v := range pkg.Versions {
		if v.Version == version {
			targetVersion = &v
			break
		}
	}

	if targetVersion == nil {
		return "", fmt.Errorf("version %s not found", version)
	}

	// Check for "all" platform first
	if url, ok := targetVersion.Downloads["all"]; ok {
		return url, nil
	}

	// Build OS/arch key
	key := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if url, ok := targetVersion.Downloads[key]; ok {
		return url, nil
	}

	return "", fmt.Errorf("no download available for %s", key)
}

// InstallProgress tracks installation progress
type InstallProgress struct {
	PackageID   string  `json:"package_id"`
	Version     string  `json:"version"`
	Status      string  `json:"status"` // downloading, extracting, configuring, complete, error
	Progress    float64 `json:"progress"`
	Message     string  `json:"message"`
	InstallPath string  `json:"install_path,omitempty"`
	Error       string  `json:"error,omitempty"`
}

// ProgressCallback is called during installation
type ProgressCallback func(progress InstallProgress)

// InstallPortablePackage downloads and installs a portable package
func InstallPortablePackage(packageID, version string, callback ProgressCallback) (*InstallProgress, error) {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}

	// Get download URL
	downloadURL, err := GetDownloadURL(pkg, version)
	if err != nil {
		return nil, err
	}

	// Setup paths
	baseDir := GetBaseDir()
	installPath := filepath.Join(baseDir, pkg.InstallPath, version)
	tempDir := filepath.Join(baseDir, ".temp")

	// Create directories
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create install directory: %w", err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	progress := InstallProgress{
		PackageID:   packageID,
		Version:     version,
		Status:      "downloading",
		Progress:    0,
		Message:     "Starting download...",
		InstallPath: installPath,
	}

	if callback != nil {
		callback(progress)
	}

	// Download file
	fileName := filepath.Base(downloadURL)
	tempFile := filepath.Join(tempDir, fileName)

	if err := downloadFile(downloadURL, tempFile, func(downloaded, total int64) {
		if total > 0 {
			progress.Progress = float64(downloaded) / float64(total) * 50 // 0-50% for download
			progress.Message = fmt.Sprintf("Downloading... %.1f%%", progress.Progress*2)
			if callback != nil {
				callback(progress)
			}
		}
	}); err != nil {
		progress.Status = "error"
		progress.Error = err.Error()
		return &progress, err
	}

	progress.Status = "extracting"
	progress.Progress = 50
	progress.Message = "Extracting files..."
	if callback != nil {
		callback(progress)
	}

	// Extract based on file type
	if err := extractArchive(tempFile, installPath); err != nil {
		progress.Status = "error"
		progress.Error = err.Error()
		return &progress, err
	}

	// Clean up temp file
	os.Remove(tempFile)

	progress.Status = "configuring"
	progress.Progress = 90
	progress.Message = "Configuring..."
	if callback != nil {
		callback(progress)
	}

	// Create default config if needed
	if pkg.ConfigFile != "" {
		createDefaultConfig(pkg, installPath)
	}

	// Record in database
	installed := models.InstalledPackage{
		PackageID:   packageID,
		Name:        pkg.Name,
		Version:     version,
		Category:    pkg.Category,
		InstallPath: installPath,
		InstalledAt: time.Now(),
		Status:      "installed",
	}
	database.DB.Create(&installed)

	progress.Status = "complete"
	progress.Progress = 100
	progress.Message = fmt.Sprintf("%s %s installed successfully", pkg.Name, version)
	if callback != nil {
		callback(progress)
	}

	return &progress, nil
}

// downloadFile downloads a file with progress tracking
func downloadFile(url, destPath string, progressFn func(downloaded, total int64)) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	total := resp.ContentLength
	var downloaded int64 = 0
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			downloaded += int64(n)
			if progressFn != nil {
				progressFn(downloaded, total)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return nil
}

// extractArchive extracts zip, tar.gz, tar.xz files
func extractArchive(archivePath, destPath string) error {
	lowerPath := strings.ToLower(archivePath)

	if strings.HasSuffix(lowerPath, ".zip") {
		return extractZip(archivePath, destPath)
	} else if strings.HasSuffix(lowerPath, ".tar.gz") || strings.HasSuffix(lowerPath, ".tgz") {
		return extractTarGz(archivePath, destPath)
	} else if strings.HasSuffix(lowerPath, ".tar.xz") {
		return extractTarXz(archivePath, destPath)
	} else if strings.HasSuffix(lowerPath, ".phar") || strings.HasSuffix(lowerPath, ".php") {
		// Single file, just copy
		return copyFile(archivePath, filepath.Join(destPath, filepath.Base(archivePath)))
	}

	return fmt.Errorf("unsupported archive format: %s", archivePath)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	// Check if all files are in a single root directory
	hasRootDir := true
	rootDirName := ""
	for _, f := range r.File {
		parts := strings.Split(f.Name, "/")
		if len(parts) == 1 && !f.FileInfo().IsDir() {
			// File at root level, no wrapping directory
			hasRootDir = false
			break
		}
		if rootDirName == "" && len(parts) > 0 {
			rootDirName = parts[0]
		} else if len(parts) > 0 && parts[0] != rootDirName {
			// Multiple root directories
			hasRootDir = false
			break
		}
	}

	for _, f := range r.File {
		fpath := f.Name

		// Strip root directory if archive has one
		if hasRootDir && rootDirName != "" {
			parts := strings.Split(f.Name, "/")
			if len(parts) > 1 {
				fpath = filepath.Join(parts[1:]...)
			} else {
				continue // Skip the root directory entry itself
			}
		}

		fpath = filepath.Join(dest, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	return extractTar(gzr, dest)
}

func extractTarXz(src, dest string) error {
	// Use xz command for .tar.xz files
	cmd := exec.Command("tar", "-xJf", src, "-C", dest, "--strip-components=1")
	return cmd.Run()
}

func extractTar(r io.Reader, dest string) error {
	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// Strip first directory component
		parts := strings.SplitN(header.Name, "/", 2)
		name := header.Name
		if len(parts) > 1 {
			name = parts[1]
		}

		target := filepath.Join(dest, name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			os.Chmod(target, os.FileMode(header.Mode))
		}
	}

	return nil
}

func copyFile(src, dest string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	destination, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func createDefaultConfig(pkg *PortablePackage, installPath string) {
	configPath := filepath.Join(installPath, pkg.ConfigFile)

	// Check if config already exists
	if _, err := os.Stat(configPath); err == nil {
		return
	}

	var configContent string

	switch pkg.ID {
	case "mysql", "mariadb":
		configContent = `[mysqld]
port=3306
datadir=./data
socket=./mysql.sock
log-error=./error.log
pid-file=./mysql.pid

[client]
port=3306
socket=./mysql.sock
`
	case "nginx":
		configContent = `worker_processes 1;

events {
    worker_connections 1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout 65;

    server {
        listen       80;
        server_name  localhost;

        location / {
            root   html;
            index  index.html index.htm index.php;
        }
    }
}
`
	}

	if configContent != "" {
		os.WriteFile(configPath, []byte(configContent), 0644)
	}
}

// GetInstalledPortablePackages returns installed packages from the file system
func GetInstalledPortablePackages() []map[string]interface{} {
	var installed []map[string]interface{}
	baseDir := GetBaseDir()

	for _, pkg := range PortableCatalog {
		pkgPath := filepath.Join(baseDir, pkg.InstallPath)
		if entries, err := os.ReadDir(pkgPath); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					versionPath := filepath.Join(pkgPath, entry.Name())
					info, _ := entry.Info()
					installed = append(installed, map[string]interface{}{
						"package_id":   pkg.ID,
						"name":         pkg.Name,
						"version":      entry.Name(),
						"category":     pkg.Category,
						"install_path": versionPath,
						"installed_at": info.ModTime(),
					})
				}
			}
		}
	}

	return installed
}

// UninstallPortablePackage removes an installed package
func UninstallPortablePackage(packageID, version string) error {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)

	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return fmt.Errorf("package not installed: %s %s", packageID, version)
	}

	// Remove directory
	if err := os.RemoveAll(installPath); err != nil {
		return err
	}

	// Remove from database
	database.DB.Where("package_id = ? AND version = ?", packageID, version).Delete(&models.InstalledPackage{})

	return nil
}

// Unused import fix
var _ = json.Marshal
