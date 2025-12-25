package handlers

import (
	"github.com/gofiber/fiber/v2"
	"vps-panel/internal/services/monitor"
)

func GetDashboard(c *fiber.Ctx) error {
	stats, err := monitor.GetSystemStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get system stats",
		})
	}

	return c.JSON(fiber.Map{
		"stats": stats,
	})
}

func GetSystemStats(c *fiber.Ctx) error {
	stats, err := monitor.GetSystemStats()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get system stats",
		})
	}

	return c.JSON(stats)
}
