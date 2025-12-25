package handlers

import (
	"fmt"
	"path/filepath"
	"vps-panel/internal/services/appstore"
	"vps-panel/internal/services/webserver"

	"github.com/gofiber/fiber/v2"
)

// GetWebServerStatus returns nginx status and PHP versions
func GetWebServerStatus(c *fiber.Ctx) error {
	status := webserver.GetNginxStatus()
	return c.JSON(status)
}

// GetSites returns all configured sites
func GetSites(c *fiber.Ctx) error {
	sites, err := webserver.GetSites()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(sites)
}

// CreateSite creates a new site
func CreateSite(c *fiber.Ctx) error {
	var site webserver.Site
	if err := c.BodyParser(&site); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if site.Name == "" || site.Domain == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Name and domain are required",
		})
	}

	// Default port
	if site.Port == 0 {
		site.Port = 80
	}

	// Default root - use /server/www/sitename
	if site.Root == "" {
		wwwDir := webserver.GetWwwDir()
		site.Root = filepath.Join(wwwDir, site.Name)
	}

	if err := webserver.CreateSite(site); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Site created successfully",
		"site":    site,
	})
}

// DeleteSite deletes a site
func DeleteSite(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Site name is required",
		})
	}

	if err := webserver.DeleteSite(name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Site deleted",
	})
}

// GetSiteConfig returns site configuration
func GetSiteConfigHandler(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Site name is required",
		})
	}

	config, err := webserver.GetSiteConfig(name)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"name":    name,
		"content": config,
	})
}

// SaveSiteConfig saves site configuration
func SaveSiteConfigHandler(c *fiber.Ctx) error {
	name := c.Params("name")

	type SaveRequest struct {
		Content string `json:"content"`
	}

	var req SaveRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := webserver.SaveSiteConfig(name, req.Content); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Configuration saved",
	})
}

// ReloadNginx reloads nginx configuration
func ReloadNginx(c *fiber.Ctx) error {
	nginxPath := webserver.GetNginxPath()
	if nginxPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "Nginx not installed",
		})
	}

	version := filepath.Base(nginxPath)
	if err := appstore.RestartService("nginx", version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":   err.Error(),
			"success": false,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Nginx reloaded",
	})
}

// GetPHPVersions returns installed PHP versions
func GetPHPVersions(c *fiber.Ctx) error {
	versions := webserver.GetInstalledPHPVersions()
	return c.JSON(versions)
}

// StartPHPCGI starts PHP-CGI server
func StartPHPCGI(c *fiber.Ctx) error {
	version := c.Query("version")
	if version == "" {
		// Use first available PHP version
		versions := webserver.GetInstalledPHPVersions()
		if len(versions) > 0 {
			version = versions[0]
		} else {
			return c.Status(404).JSON(fiber.Map{
				"error": "No PHP version installed",
			})
		}
	}

	port := 9000 // Default FastCGI port

	if err := webserver.StartPHPCGI(version, port); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": fmt.Sprintf("PHP-CGI %s started on port %d", version, port),
		"version": version,
		"port":    port,
	})
}

// StopPHPCGI stops PHP-CGI server
func StopPHPCGI(c *fiber.Ctx) error {
	if err := webserver.StopPHPCGI(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "PHP-CGI stopped",
	})
}

// GetPHPCGIStatus returns PHP-CGI status
func GetPHPCGIStatus(c *fiber.Ctx) error {
	running := webserver.IsPHPCGIRunning()
	return c.JSON(fiber.Map{
		"running": running,
	})
}
