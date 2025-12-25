package firewall

import (
	"fmt"
	"os/exec"
	"runtime"

	"vps-panel/internal/database"
	"vps-panel/internal/models"
)

// GetRules returns list of firewall rules from DB
func GetRules() ([]models.FirewallRule, error) {
	var rules []models.FirewallRule
	err := database.DB.Order("created_at desc").Find(&rules).Error
	return rules, err
}

// AddRule adds a firewall rule
func AddRule(name, port, protocol, action string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("platform not supported")
	}

	// Check if exists in DB
	var count int64
	database.DB.Model(&models.FirewallRule{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return fmt.Errorf("rule with name '%s' already exists", name)
	}

	// netsh advfirewall firewall add rule name="Open Port 80" dir=in action=allow protocol=TCP localport=80
	cmd := exec.Command("netsh", "advfirewall", "firewall", "add", "rule",
		fmt.Sprintf("name=%s", name),
		"dir=in",
		fmt.Sprintf("action=%s", action),
		fmt.Sprintf("protocol=%s", protocol),
		fmt.Sprintf("localport=%s", port),
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to add rule: %v", err)
	}

	// Save to DB
	rule := models.FirewallRule{
		Name:     name,
		Port:     port,
		Protocol: protocol,
		Action:   action,
	}
	return database.DB.Create(&rule).Error
}

// DeleteRule deletes a firewall rule
func DeleteRule(name string) error {
	if runtime.GOOS != "windows" {
		return fmt.Errorf("platform not supported")
	}

	// netsh advfirewall firewall delete rule name="Open Port 80"
	cmd := exec.Command("netsh", "advfirewall", "firewall", "delete", "rule", fmt.Sprintf("name=%s", name))
	if err := cmd.Run(); err != nil {
		// Even if it fails (e.g. not found in netsh), we should remove from DB if it exists there
		// But maybe better to return error? Let's proceed to delete from DB anyway to keep sync.
	}

	return database.DB.Where("name = ?", name).Delete(&models.FirewallRule{}).Error
}
