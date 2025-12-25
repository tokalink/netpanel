package handlers

import (
	"vps-panel/internal/services/appstore"

	"github.com/gofiber/fiber/v2"
)

// GetAllServices returns all installed services with their status
func GetAllServices(c *fiber.Ctx) error {
	installed := appstore.GetInstalledPortablePackages()

	var services []map[string]interface{}

	for _, inst := range installed {
		pkgID := inst["package_id"].(string)
		version := inst["version"].(string)

		status, err := appstore.GetServiceStatus(pkgID, version)
		if err != nil {
			continue
		}

		services = append(services, map[string]interface{}{
			"package_id":   pkgID,
			"name":         status.Name,
			"version":      version,
			"running":      status.Running,
			"port":         status.Port,
			"install_path": status.InstallPath,
			"config_path":  status.ConfigPath,
			"category":     inst["category"],
		})
	}

	return c.JSON(services)
}

// ServiceAction handles start/stop/restart for a service
func ServiceAction(c *fiber.Ctx) error {
	packageID := c.Params("id")
	action := c.Params("action")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	var err error
	var message string

	switch action {
	case "start":
		err = appstore.StartService(packageID, version)
		message = "Service started"
	case "stop":
		err = appstore.StopService(packageID, version)
		message = "Service stopped"
	case "restart":
		err = appstore.RestartService(packageID, version)
		message = "Service restarted"
	default:
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid action. Use: start, stop, restart",
		})
	}

	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": message,
	})
}

// GetServiceLogs returns log file content
func GetServiceLogs(c *fiber.Ctx) error {
	packageID := c.Params("id")
	version := c.Query("version")

	if packageID == "" || version == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Package ID and version are required",
		})
	}

	content, err := appstore.GetLog(packageID, version)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"log": content,
	})
}
