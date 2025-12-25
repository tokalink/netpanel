package handlers

import (
	"strconv"
	"vps-panel/internal/services/cron"

	"github.com/gofiber/fiber/v2"
)

// GetCronJobs returns all cron jobs
func GetCronJobs(c *fiber.Ctx) error {
	jobs, err := cron.GetJobs()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(jobs)
}

// AddCronJob adds a new cron job
func AddCronJob(c *fiber.Ctx) error {
	type Request struct {
		Name     string `json:"name"`
		Schedule string `json:"schedule"`
		Command  string `json:"command"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Schedule == "" || req.Command == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Name, schedule and command are required",
		})
	}

	job, err := cron.AddJob(req.Name, req.Schedule, req.Command)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(job)
}

// RemoveCronJob removes a cron job
func RemoveCronJob(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	if err := cron.RemoveJob(uint(id)); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}

// ToggleCronJob enables or disables a job
func ToggleCronJob(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}

	type Request struct {
		Enabled bool `json:"enabled"`
	}
	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if err := cron.ToggleJob(uint(id), req.Enabled); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
	})
}
