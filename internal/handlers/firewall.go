package handlers

import (
	"vps-panel/internal/services/firewall"

	"github.com/gofiber/fiber/v2"
)

// GetFirewallRules returns list of firewall rules
func GetFirewallRules(c *fiber.Ctx) error {
	rules, err := firewall.GetRules()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(rules)
}

// AddFirewallRule adds a firewall rule
func AddFirewallRule(c *fiber.Ctx) error {
	type Request struct {
		Name     string `json:"name"`
		Port     string `json:"port"`
		Protocol string `json:"protocol"`
		Action   string `json:"action"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" || req.Port == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Name and port are required",
		})
	}

	if req.Protocol == "" {
		req.Protocol = "TCP"
	}
	if req.Action == "" {
		req.Action = "allow"
	}

	if err := firewall.AddRule(req.Name, req.Port, req.Protocol, req.Action); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Rule added successfully",
	})
}

// DeleteFirewallRule deletes a firewall rule
func DeleteFirewallRule(c *fiber.Ctx) error {
	type Request struct {
		Name string `json:"name"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Rule name is required",
		})
	}

	if err := firewall.DeleteRule(req.Name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Rule deleted",
	})
}
