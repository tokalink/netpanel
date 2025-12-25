package handlers

import (
	"vps-panel/internal/services/appstore"

	"github.com/gofiber/fiber/v2"
)

// GetPackages returns all available packages
func GetPackages(c *fiber.Ctx) error {
	category := c.Query("category", "all")
	packages := appstore.GetPackagesByCategory(category)

	// Add installed status to each package
	result := make([]map[string]interface{}, len(packages))
	for i, pkg := range packages {
		result[i] = map[string]interface{}{
			"id":          pkg.ID,
			"name":        pkg.Name,
			"description": pkg.Description,
			"category":    pkg.Category,
			"icon":        pkg.Icon,
			"versions":    pkg.Versions,
			"service":     pkg.Service,
			"ports":       pkg.Ports,
			"installed":   appstore.IsPackageInstalled(pkg.ID),
		}
	}

	return c.JSON(result)
}

// GetPackageStatus returns the status of a specific package
func GetPackageStatus(c *fiber.Ctx) error {
	packageID := c.Params("id")
	status := appstore.CheckPackageStatus(packageID)
	status["db_installed"] = appstore.IsPackageInstalled(packageID)
	return c.JSON(status)
}

// GetInstalledPackages returns all installed packages
func GetInstalledPackages(c *fiber.Ctx) error {
	packages, err := appstore.GetInstalledPackages()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get installed packages",
		})
	}
	return c.JSON(packages)
}

// InstallPackage handles package installation requests
func InstallPackage(c *fiber.Ctx) error {
	type InstallRequest struct {
		PackageID string `json:"package_id"`
		Version   string `json:"version"`
	}

	var req InstallRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.PackageID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID is required",
		})
	}

	// Get package info
	pkg := appstore.GetPackageByID(req.PackageID)
	if pkg == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package not found",
		})
	}

	// Check package manager
	pm := appstore.DetectPackageManager()
	if pm == "none" {
		return c.Status(400).JSON(fiber.Map{
			"error":   "No package manager available",
			"message": "Please install Chocolatey (Windows), apt/dnf (Linux), or Homebrew (macOS)",
		})
	}

	// Get install command for preview
	cmd, err := appstore.GetInstallCommand(req.PackageID, req.Version)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Perform installation
	result, err := appstore.InstallPackage(req.PackageID, req.Version)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success":         result.Success,
		"message":         result.Message,
		"output":          result.Output,
		"command":         cmd,
		"package_manager": pm,
	})
}

// UninstallPackage handles package uninstallation requests
func UninstallPackage(c *fiber.Ctx) error {
	packageID := c.Params("id")

	if packageID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID is required",
		})
	}

	result, err := appstore.UninstallPackage(packageID)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": result.Success,
		"message": result.Message,
		"output":  result.Output,
	})
}

// GetSystemInfo returns system package manager info
func GetSystemInfo(c *fiber.Ctx) error {
	pm := appstore.DetectPackageManager()

	return c.JSON(fiber.Map{
		"package_manager": pm,
		"os":              c.Get("User-Agent"),
	})
}

// PreviewInstall returns the install command without executing
func PreviewInstall(c *fiber.Ctx) error {
	type PreviewRequest struct {
		PackageID string `json:"package_id"`
		Version   string `json:"version"`
	}

	var req PreviewRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	pm := appstore.DetectPackageManager()
	cmd, err := appstore.GetInstallCommand(req.PackageID, req.Version)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"command":         cmd,
		"package_manager": pm,
	})
}
