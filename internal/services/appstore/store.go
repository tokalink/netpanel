package appstore

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"vps-panel/internal/database"
	"vps-panel/internal/models"
)

// Package represents an installable software package
type Package struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Category    string           `json:"category"`
	Icon        string           `json:"icon"`
	Versions    []PackageVersion `json:"versions"`
	Service     string           `json:"service,omitempty"`
	Ports       []int            `json:"ports,omitempty"`
}

type PackageVersion struct {
	Version string `json:"version"`
	Latest  bool   `json:"latest,omitempty"`
	LTS     bool   `json:"lts,omitempty"`
}

// InstallCommand contains OS-specific install commands
type InstallCommand struct {
	Windows WindowsInstall `json:"windows"`
	Linux   LinuxInstall   `json:"linux"`
	Darwin  DarwinInstall  `json:"darwin"`
}

type WindowsInstall struct {
	Choco  string `json:"choco,omitempty"`
	Winget string `json:"winget,omitempty"`
	Script string `json:"script,omitempty"`
}

type LinuxInstall struct {
	Apt    string `json:"apt,omitempty"`
	Yum    string `json:"yum,omitempty"`
	Dnf    string `json:"dnf,omitempty"`
	Script string `json:"script,omitempty"`
}

type DarwinInstall struct {
	Brew   string `json:"brew,omitempty"`
	Script string `json:"script,omitempty"`
}

// PackageCatalog holds all available packages
var PackageCatalog = []Package{
	{
		ID:          "mysql",
		Name:        "MySQL Server",
		Description: "Open-source relational database management system",
		Category:    "database",
		Icon:        "database",
		Service:     "mysql",
		Ports:       []int{3306},
		Versions: []PackageVersion{
			{Version: "8.0", Latest: true},
			{Version: "5.7"},
		},
	},
	{
		ID:          "mariadb",
		Name:        "MariaDB",
		Description: "Community-developed fork of MySQL",
		Category:    "database",
		Icon:        "database",
		Service:     "mariadb",
		Ports:       []int{3306},
		Versions: []PackageVersion{
			{Version: "11.0", Latest: true},
			{Version: "10.11", LTS: true},
			{Version: "10.6"},
		},
	},
	{
		ID:          "postgresql",
		Name:        "PostgreSQL",
		Description: "Advanced open-source relational database",
		Category:    "database",
		Icon:        "database",
		Service:     "postgresql",
		Ports:       []int{5432},
		Versions: []PackageVersion{
			{Version: "16", Latest: true},
			{Version: "15"},
			{Version: "14"},
		},
	},
	{
		ID:          "redis",
		Name:        "Redis",
		Description: "In-memory data structure store and cache",
		Category:    "database",
		Icon:        "database",
		Service:     "redis",
		Ports:       []int{6379},
		Versions: []PackageVersion{
			{Version: "7.2", Latest: true},
			{Version: "7.0"},
			{Version: "6.2"},
		},
	},
	{
		ID:          "mongodb",
		Name:        "MongoDB",
		Description: "Document-oriented NoSQL database",
		Category:    "database",
		Icon:        "database",
		Service:     "mongod",
		Ports:       []int{27017},
		Versions: []PackageVersion{
			{Version: "7.0", Latest: true},
			{Version: "6.0"},
		},
	},
	{
		ID:          "nginx",
		Name:        "Nginx",
		Description: "High-performance web server and reverse proxy",
		Category:    "webserver",
		Icon:        "globe",
		Service:     "nginx",
		Ports:       []int{80, 443},
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
			{Version: "mainline"},
		},
	},
	{
		ID:          "apache",
		Name:        "Apache HTTP Server",
		Description: "Popular open-source web server",
		Category:    "webserver",
		Icon:        "globe",
		Service:     "apache2",
		Ports:       []int{80, 443},
		Versions: []PackageVersion{
			{Version: "2.4", Latest: true},
		},
	},
	{
		ID:          "php",
		Name:        "PHP",
		Description: "Popular server-side scripting language",
		Category:    "runtime",
		Icon:        "code",
		Versions: []PackageVersion{
			{Version: "8.3", Latest: true},
			{Version: "8.2"},
			{Version: "8.1"},
			{Version: "8.0"},
			{Version: "7.4"},
		},
	},
	{
		ID:          "nodejs",
		Name:        "Node.js",
		Description: "JavaScript runtime built on Chrome's V8 engine",
		Category:    "runtime",
		Icon:        "code",
		Versions: []PackageVersion{
			{Version: "20", Latest: true, LTS: true},
			{Version: "18", LTS: true},
			{Version: "21"},
		},
	},
	{
		ID:          "python",
		Name:        "Python",
		Description: "High-level programming language",
		Category:    "runtime",
		Icon:        "code",
		Versions: []PackageVersion{
			{Version: "3.12", Latest: true},
			{Version: "3.11"},
			{Version: "3.10"},
		},
	},
	{
		ID:          "go",
		Name:        "Go",
		Description: "Statically typed, compiled programming language",
		Category:    "runtime",
		Icon:        "code",
		Versions: []PackageVersion{
			{Version: "1.22", Latest: true},
			{Version: "1.21"},
		},
	},
	{
		ID:          "docker",
		Name:        "Docker",
		Description: "Container runtime platform",
		Category:    "tools",
		Icon:        "box",
		Service:     "docker",
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
		},
	},
	{
		ID:          "fail2ban",
		Name:        "Fail2Ban",
		Description: "Intrusion prevention software",
		Category:    "security",
		Icon:        "shield",
		Service:     "fail2ban",
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
		},
	},
	{
		ID:          "certbot",
		Name:        "Certbot",
		Description: "Let's Encrypt SSL certificate tool",
		Category:    "security",
		Icon:        "lock",
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
		},
	},
	{
		ID:          "git",
		Name:        "Git",
		Description: "Distributed version control system",
		Category:    "tools",
		Icon:        "git-branch",
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
		},
	},
	{
		ID:          "composer",
		Name:        "Composer",
		Description: "PHP dependency manager",
		Category:    "tools",
		Icon:        "package",
		Versions: []PackageVersion{
			{Version: "latest", Latest: true},
		},
	},
}

// GetPackages returns all available packages
func GetPackages() []Package {
	return PackageCatalog
}

// GetPackageByID returns a package by its ID
func GetPackageByID(id string) *Package {
	for _, pkg := range PackageCatalog {
		if pkg.ID == id {
			return &pkg
		}
	}
	return nil
}

// GetPackagesByCategory returns packages filtered by category
func GetPackagesByCategory(category string) []Package {
	if category == "" || category == "all" {
		return PackageCatalog
	}

	var result []Package
	for _, pkg := range PackageCatalog {
		if pkg.Category == category {
			result = append(result, pkg)
		}
	}
	return result
}

// GetInstalledPackages returns all installed packages from database
func GetInstalledPackages() ([]models.InstalledPackage, error) {
	var packages []models.InstalledPackage
	if err := database.DB.Find(&packages).Error; err != nil {
		return nil, err
	}
	return packages, nil
}

// IsPackageInstalled checks if a package is already installed
func IsPackageInstalled(packageID string) bool {
	var count int64
	database.DB.Model(&models.InstalledPackage{}).Where("package_id = ?", packageID).Count(&count)
	return count > 0
}

// InstallResult represents the result of an installation
type InstallResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Output  string `json:"output,omitempty"`
}

// DetectPackageManager detects the available package manager
func DetectPackageManager() string {
	switch runtime.GOOS {
	case "windows":
		// Check for chocolatey
		if _, err := exec.LookPath("choco"); err == nil {
			return "choco"
		}
		// Check for winget
		if _, err := exec.LookPath("winget"); err == nil {
			return "winget"
		}
		return "none"
	case "linux":
		// Check for apt
		if _, err := exec.LookPath("apt"); err == nil {
			return "apt"
		}
		// Check for dnf
		if _, err := exec.LookPath("dnf"); err == nil {
			return "dnf"
		}
		// Check for yum
		if _, err := exec.LookPath("yum"); err == nil {
			return "yum"
		}
		return "none"
	case "darwin":
		// Check for brew
		if _, err := exec.LookPath("brew"); err == nil {
			return "brew"
		}
		return "none"
	}
	return "none"
}

// GetInstallCommand returns the install command for a package
func GetInstallCommand(packageID, version string) (string, error) {
	pm := DetectPackageManager()
	if pm == "none" {
		return "", fmt.Errorf("no supported package manager found")
	}

	pkg := GetPackageByID(packageID)
	if pkg == nil {
		return "", fmt.Errorf("package not found: %s", packageID)
	}

	// Build install command based on package manager
	switch pm {
	case "choco":
		return getChocoCommand(packageID, version), nil
	case "winget":
		return getWingetCommand(packageID, version), nil
	case "apt":
		return getAptCommand(packageID, version), nil
	case "dnf", "yum":
		return getDnfCommand(packageID, version, pm), nil
	case "brew":
		return getBrewCommand(packageID, version), nil
	}

	return "", fmt.Errorf("unsupported package manager: %s", pm)
}

func getChocoCommand(packageID, version string) string {
	pkgMap := map[string]string{
		"mysql":      "mysql",
		"mariadb":    "mariadb",
		"postgresql": "postgresql",
		"redis":      "redis",
		"mongodb":    "mongodb",
		"nginx":      "nginx",
		"php":        "php",
		"nodejs":     "nodejs-lts",
		"python":     "python",
		"go":         "golang",
		"docker":     "docker-desktop",
		"git":        "git",
		"composer":   "composer",
	}

	pkgName := pkgMap[packageID]
	if pkgName == "" {
		pkgName = packageID
	}

	if version != "" && version != "latest" {
		return fmt.Sprintf("choco install %s --version=%s -y", pkgName, version)
	}
	return fmt.Sprintf("choco install %s -y", pkgName)
}

func getWingetCommand(packageID, version string) string {
	pkgMap := map[string]string{
		"mysql":      "Oracle.MySQL",
		"postgresql": "PostgreSQL.PostgreSQL",
		"nodejs":     "OpenJS.NodeJS.LTS",
		"python":     "Python.Python.3.12",
		"go":         "GoLang.Go",
		"docker":     "Docker.DockerDesktop",
		"git":        "Git.Git",
	}

	pkgName := pkgMap[packageID]
	if pkgName == "" {
		return ""
	}

	return fmt.Sprintf("winget install --id=%s -e --accept-package-agreements --accept-source-agreements", pkgName)
}

func getAptCommand(packageID, version string) string {
	pkgMap := map[string]string{
		"mysql":      "mysql-server",
		"mariadb":    "mariadb-server",
		"postgresql": "postgresql",
		"redis":      "redis-server",
		"mongodb":    "mongodb",
		"nginx":      "nginx",
		"apache":     "apache2",
		"php":        "php",
		"python":     "python3",
		"go":         "golang",
		"fail2ban":   "fail2ban",
		"certbot":    "certbot",
		"git":        "git",
		"docker":     "docker.io",
	}

	pkgName := pkgMap[packageID]
	if pkgName == "" {
		pkgName = packageID
	}

	// Handle PHP with version
	if packageID == "php" && version != "" && version != "latest" {
		return fmt.Sprintf("apt-get install -y php%s php%s-fpm php%s-cli php%s-common php%s-mysql php%s-curl php%s-mbstring php%s-xml",
			version, version, version, version, version, version, version, version)
	}

	// Handle Node.js with NodeSource
	if packageID == "nodejs" {
		return fmt.Sprintf("curl -fsSL https://deb.nodesource.com/setup_%s.x | bash - && apt-get install -y nodejs", version)
	}

	return fmt.Sprintf("apt-get install -y %s", pkgName)
}

func getDnfCommand(packageID, version, pm string) string {
	pkgMap := map[string]string{
		"mysql":      "mysql-server",
		"mariadb":    "mariadb-server",
		"postgresql": "postgresql-server",
		"redis":      "redis",
		"nginx":      "nginx",
		"apache":     "httpd",
		"php":        "php",
		"python":     "python3",
		"go":         "golang",
		"fail2ban":   "fail2ban",
		"git":        "git",
		"docker":     "docker",
	}

	pkgName := pkgMap[packageID]
	if pkgName == "" {
		pkgName = packageID
	}

	return fmt.Sprintf("%s install -y %s", pm, pkgName)
}

func getBrewCommand(packageID, version string) string {
	pkgMap := map[string]string{
		"mysql":      "mysql",
		"mariadb":    "mariadb",
		"postgresql": "postgresql",
		"redis":      "redis",
		"mongodb":    "mongodb-community",
		"nginx":      "nginx",
		"php":        "php",
		"nodejs":     "node",
		"python":     "python",
		"go":         "go",
		"git":        "git",
		"composer":   "composer",
	}

	pkgName := pkgMap[packageID]
	if pkgName == "" {
		pkgName = packageID
	}

	if version != "" && version != "latest" {
		return fmt.Sprintf("brew install %s@%s", pkgName, version)
	}
	return fmt.Sprintf("brew install %s", pkgName)
}

// InstallPackage installs a package
func InstallPackage(packageID, version string) (*InstallResult, error) {
	// Check if already installed
	if IsPackageInstalled(packageID) {
		return &InstallResult{
			Success: false,
			Message: "Package is already installed",
		}, nil
	}

	pkg := GetPackageByID(packageID)
	if pkg == nil {
		return &InstallResult{
			Success: false,
			Message: "Package not found",
		}, nil
	}

	// Get install command
	cmd, err := GetInstallCommand(packageID, version)
	if err != nil {
		return &InstallResult{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	// Execute installation
	var output []byte
	var execErr error

	switch runtime.GOOS {
	case "windows":
		output, execErr = exec.Command("powershell", "-Command", cmd).CombinedOutput()
	default:
		output, execErr = exec.Command("bash", "-c", cmd).CombinedOutput()
	}

	if execErr != nil {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Installation failed: %v", execErr),
			Output:  string(output),
		}, nil
	}

	// Record installation in database
	installed := models.InstalledPackage{
		PackageID:   packageID,
		Name:        pkg.Name,
		Version:     version,
		Category:    pkg.Category,
		InstalledAt: time.Now(),
		Status:      "installed",
	}

	if err := database.DB.Create(&installed).Error; err != nil {
		return &InstallResult{
			Success: true,
			Message: "Package installed but failed to record in database",
			Output:  string(output),
		}, nil
	}

	return &InstallResult{
		Success: true,
		Message: fmt.Sprintf("%s installed successfully", pkg.Name),
		Output:  string(output),
	}, nil
}

// UninstallPackage removes an installed package
func UninstallPackage(packageID string) (*InstallResult, error) {
	if !IsPackageInstalled(packageID) {
		return &InstallResult{
			Success: false,
			Message: "Package is not installed",
		}, nil
	}

	pm := DetectPackageManager()
	pkg := GetPackageByID(packageID)
	if pkg == nil {
		return &InstallResult{
			Success: false,
			Message: "Package not found",
		}, nil
	}

	var cmd string
	switch pm {
	case "choco":
		cmd = fmt.Sprintf("choco uninstall %s -y", packageID)
	case "apt":
		cmd = fmt.Sprintf("apt-get remove -y %s", packageID)
	case "dnf", "yum":
		cmd = fmt.Sprintf("%s remove -y %s", pm, packageID)
	case "brew":
		cmd = fmt.Sprintf("brew uninstall %s", packageID)
	default:
		return &InstallResult{
			Success: false,
			Message: "No package manager available",
		}, nil
	}

	var output []byte
	var execErr error

	switch runtime.GOOS {
	case "windows":
		output, execErr = exec.Command("powershell", "-Command", cmd).CombinedOutput()
	default:
		output, execErr = exec.Command("bash", "-c", cmd).CombinedOutput()
	}

	if execErr != nil {
		return &InstallResult{
			Success: false,
			Message: fmt.Sprintf("Uninstallation failed: %v", execErr),
			Output:  string(output),
		}, nil
	}

	// Remove from database
	database.DB.Where("package_id = ?", packageID).Delete(&models.InstalledPackage{})

	return &InstallResult{
		Success: true,
		Message: fmt.Sprintf("%s uninstalled successfully", pkg.Name),
		Output:  string(output),
	}, nil
}

// CheckPackageStatus verifies if a package is actually installed on the system
func CheckPackageStatus(packageID string) map[string]interface{} {
	result := map[string]interface{}{
		"id":        packageID,
		"installed": false,
		"version":   "",
	}

	pkg := GetPackageByID(packageID)
	if pkg == nil {
		return result
	}

	// Check if the package binary or service exists
	var checkCmd string
	switch runtime.GOOS {
	case "windows":
		switch packageID {
		case "nodejs":
			checkCmd = "node --version"
		case "python":
			checkCmd = "python --version"
		case "php":
			checkCmd = "php --version"
		case "go":
			checkCmd = "go version"
		case "git":
			checkCmd = "git --version"
		case "docker":
			checkCmd = "docker --version"
		default:
			checkCmd = fmt.Sprintf("where %s", packageID)
		}
	default:
		switch packageID {
		case "nodejs":
			checkCmd = "node --version 2>/dev/null"
		case "python":
			checkCmd = "python3 --version 2>/dev/null"
		case "php":
			checkCmd = "php --version 2>/dev/null | head -1"
		case "go":
			checkCmd = "go version 2>/dev/null"
		case "git":
			checkCmd = "git --version 2>/dev/null"
		case "docker":
			checkCmd = "docker --version 2>/dev/null"
		case "nginx":
			checkCmd = "nginx -v 2>&1"
		case "mysql":
			checkCmd = "mysql --version 2>/dev/null"
		case "postgresql":
			checkCmd = "psql --version 2>/dev/null"
		case "redis":
			checkCmd = "redis-server --version 2>/dev/null"
		default:
			checkCmd = fmt.Sprintf("which %s 2>/dev/null", packageID)
		}
	}

	var output []byte
	var err error

	switch runtime.GOOS {
	case "windows":
		output, err = exec.Command("powershell", "-Command", checkCmd).CombinedOutput()
	default:
		output, err = exec.Command("bash", "-c", checkCmd).CombinedOutput()
	}

	if err == nil && len(output) > 0 {
		result["installed"] = true
		result["version"] = strings.TrimSpace(string(output))
	}

	return result
}

// Ensure json import is used
var _ = json.Marshal
