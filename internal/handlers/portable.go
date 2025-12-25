package handlers

import (
	"vps-panel/internal/services/appstore"

	"github.com/gofiber/fiber/v2"
)

// GetPortablePackages returns all available portable packages
func GetPortablePackages(c *fiber.Ctx) error {
	category := c.Query("category", "all")
	packages := appstore.GetPortablePackages()

	// Filter by category if specified
	if category != "all" {
		var filtered []appstore.PortablePackage
		for _, pkg := range packages {
			if pkg.Category == category {
				filtered = append(filtered, pkg)
			}
		}
		packages = filtered
	}

	// Check installed status for each version
	result := make([]map[string]interface{}, len(packages))
	installed := appstore.GetInstalledPortablePackages()

	for i, pkg := range packages {
		versions := make([]map[string]interface{}, len(pkg.Versions))
		for j, ver := range pkg.Versions {
			isInstalled := false
			isRunning := false
			for _, inst := range installed {
				if inst["package_id"] == pkg.ID && inst["version"] == ver.Version {
					isInstalled = true
					// Check if running
					status, err := appstore.GetServiceStatus(pkg.ID, ver.Version)
					if err == nil && status.Running {
						isRunning = true
					}
					break
				}
			}
			versions[j] = map[string]interface{}{
				"version":   ver.Version,
				"latest":    ver.Latest,
				"lts":       ver.LTS,
				"installed": isInstalled,
				"running":   isRunning,
			}
		}

		result[i] = map[string]interface{}{
			"id":           pkg.ID,
			"name":         pkg.Name,
			"description":  pkg.Description,
			"category":     pkg.Category,
			"versions":     versions,
			"install_path": pkg.InstallPath,
			"ports":        pkg.Ports,
		}
	}

	return c.JSON(result)
}

// GetPortableInstalled returns installed portable packages
func GetPortableInstalled(c *fiber.Ctx) error {
	installed := appstore.GetInstalledPortablePackages()
	return c.JSON(installed)
}

// InstallPortablePackage handles portable package installation
func InstallPortablePackage(c *fiber.Ctx) error {
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

	if req.PackageID == "" || req.Version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	// Get package info
	pkg := appstore.GetPortablePackageByID(req.PackageID)
	if pkg == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package not found",
		})
	}

	// Get download URL to verify availability
	downloadURL, err := appstore.GetDownloadURL(pkg, req.Version)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "This package/version is not available for your platform",
		})
	}

	// Perform installation
	result, err := appstore.InstallPortablePackage(req.PackageID, req.Version, nil)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"message": "Installation failed",
		})
	}

	return c.JSON(fiber.Map{
		"success":      result.Status == "complete",
		"message":      result.Message,
		"status":       result.Status,
		"install_path": result.InstallPath,
		"download_url": downloadURL,
	})
}

// UninstallPortablePackage handles package removal
func UninstallPortablePackage(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	if err := appstore.UninstallPortablePackage(packageID, version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Package uninstalled successfully",
	})
}

// GetPortableSystemInfo returns system info for portable packages
func GetPortableSystemInfo(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"base_dir":   appstore.GetBaseDir(),
		"mode":       "portable",
		"categories": []string{"database", "runtime", "webserver", "tools"},
	})
}

// PreviewPortableInstall returns info about the install without executing
func PreviewPortableInstall(c *fiber.Ctx) error {
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

	pkg := appstore.GetPortablePackageByID(req.PackageID)
	if pkg == nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Package not found",
		})
	}

	downloadURL, err := appstore.GetDownloadURL(pkg, req.Version)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	installPath := appstore.GetBaseDir() + "/" + pkg.InstallPath + "/" + req.Version

	return c.JSON(fiber.Map{
		"package_id":   req.PackageID,
		"version":      req.Version,
		"download_url": downloadURL,
		"install_path": installPath,
		"mode":         "portable",
	})
}

// GetServiceStatus returns status of an installed service
func GetServiceStatus(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	status, err := appstore.GetServiceStatus(packageID, version)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(status)
}

// StartService starts an installed service
func StartService(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	if err := appstore.StartService(packageID, version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service started",
	})
}

// StopService stops a running service
func StopService(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	if err := appstore.StopService(packageID, version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service stopped",
	})
}

// RestartService restarts a service
func RestartService(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	if err := appstore.RestartService(packageID, version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Service restarted",
	})
}

// GetServiceConfig returns configuration file content
func GetServiceConfig(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	configPath, content, err := appstore.GetConfig(packageID, version)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"config_path": configPath,
		"content":     content,
	})
}

// SaveServiceConfig saves configuration file
func SaveServiceConfig(c *fiber.Ctx) error {
	packageID := c.Params("id")

	type SaveRequest struct {
		Version string `json:"version"`
		Content string `json:"content"`
	}

	var req SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := appstore.SaveConfig(packageID, req.Version, req.Content); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Configuration saved",
	})
}
