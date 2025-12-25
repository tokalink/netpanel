package webserver

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"vps-panel/internal/services/appstore"
)

// Site represents a website/virtual host configuration
type Site struct {
	Name       string `json:"name"`
	Domain     string `json:"domain"`
	Port       int    `json:"port"`
	Root       string `json:"root"`
	PHPVersion string `json:"php_version,omitempty"`
	SSL        bool   `json:"ssl"`
	Enabled    bool   `json:"enabled"`
	ConfigPath string `json:"config_path"`
}

// GetNginxPath returns the path to Nginx installation
func GetNginxPath() string {
	baseDir := appstore.GetBaseDir()
	nginxDir := filepath.Join(baseDir, "webserver", "nginx")

	// Find installed version
	entries, err := os.ReadDir(nginxDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(nginxDir, entry.Name())
		}
	}
	return ""
}

// GetWwwDir returns the default www directory for sites
func GetWwwDir() string {
	baseDir := appstore.GetBaseDir()
	wwwDir := filepath.Join(baseDir, "www")
	os.MkdirAll(wwwDir, 0755)
	return wwwDir
}

// GetInstalledPHPVersions returns all installed PHP versions
func GetInstalledPHPVersions() []string {
	var versions []string
	baseDir := appstore.GetBaseDir()
	phpDir := filepath.Join(baseDir, "runtime", "php")

	entries, err := os.ReadDir(phpDir)
	if err != nil {
		return versions
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if php executable exists
			phpExe := "php"
			if runtime.GOOS == "windows" {
				phpExe = "php.exe"
			}
			if _, err := os.Stat(filepath.Join(phpDir, entry.Name(), phpExe)); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}
	return versions
}

// GetPHPCGIPath returns the path to PHP-CGI executable
func GetPHPCGIPath(version string) string {
	baseDir := appstore.GetBaseDir()
	if runtime.GOOS == "windows" {
		return filepath.Join(baseDir, "runtime", "php", version, "php-cgi.exe")
	}
	return filepath.Join(baseDir, "runtime", "php", version, "bin", "php-cgi")
}

// StartPHPCGI starts PHP-CGI FastCGI server on specified port
func StartPHPCGI(version string, port int) error {
	phpCgiPath := GetPHPCGIPath(version)
	if _, err := os.Stat(phpCgiPath); os.IsNotExist(err) {
		return fmt.Errorf("PHP-CGI not found: %s", phpCgiPath)
	}

	baseDir := appstore.GetBaseDir()
	phpDir := filepath.Join(baseDir, "runtime", "php", version)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: use php-cgi with -b flag for FastCGI
		cmd = exec.Command(phpCgiPath, "-b", fmt.Sprintf("127.0.0.1:%d", port))
	} else {
		// Linux/Mac: spawn-fcgi or php-cgi -b
		cmd = exec.Command(phpCgiPath, "-b", fmt.Sprintf("127.0.0.1:%d", port))
	}

	cmd.Dir = phpDir

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start PHP-CGI: %w", err)
	}

	return nil
}

// StopPHPCGI stops PHP-CGI processes
func StopPHPCGI() error {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("taskkill", "/F", "/IM", "php-cgi.exe")
	} else {
		cmd = exec.Command("pkill", "-9", "php-cgi")
	}
	return cmd.Run()
}

// IsPHPCGIRunning checks if PHP-CGI is running
func IsPHPCGIRunning() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq php-cgi.exe")
	} else {
		cmd = exec.Command("pgrep", "php-cgi")
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return strings.Contains(string(output), "php-cgi")
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetSitesDir returns the directory for site configs
func GetSitesDir() string {
	nginxPath := GetNginxPath()
	if nginxPath == "" {
		return ""
	}
	sitesDir := filepath.Join(nginxPath, "conf", "sites")
	os.MkdirAll(sitesDir, 0755)
	return sitesDir
}

// GetSites returns all configured sites
func GetSites() ([]Site, error) {
	sitesDir := GetSitesDir()
	if sitesDir == "" {
		return nil, fmt.Errorf("nginx not installed")
	}

	var sites []Site

	entries, err := os.ReadDir(sitesDir)
	if err != nil {
		return sites, nil
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".conf") {
			site, err := parseSiteConfig(filepath.Join(sitesDir, entry.Name()))
			if err == nil {
				sites = append(sites, site)
			}
		}
	}

	return sites, nil
}

// parseSiteConfig parses a nginx site config file
func parseSiteConfig(configPath string) (Site, error) {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return Site{}, err
	}

	site := Site{
		ConfigPath: configPath,
		Name:       strings.TrimSuffix(filepath.Base(configPath), ".conf"),
		Enabled:    true,
		Port:       80,
	}

	text := string(content)

	// Parse server_name
	if match := regexp.MustCompile(`server_name\s+([^;]+);`).FindStringSubmatch(text); len(match) > 1 {
		site.Domain = strings.TrimSpace(match[1])
	}

	// Parse listen port
	if match := regexp.MustCompile(`listen\s+(\d+)`).FindStringSubmatch(text); len(match) > 1 {
		fmt.Sscanf(match[1], "%d", &site.Port)
	}

	// Parse root
	if match := regexp.MustCompile(`root\s+([^;]+);`).FindStringSubmatch(text); len(match) > 1 {
		site.Root = strings.TrimSpace(match[1])
	}

	// Check SSL
	site.SSL = strings.Contains(text, "ssl_certificate")

	// Parse PHP version from fastcgi_pass comment or path
	if match := regexp.MustCompile(`# PHP Version: ([^\n]+)`).FindStringSubmatch(text); len(match) > 1 {
		site.PHPVersion = strings.TrimSpace(match[1])
	}

	return site, nil
}

// CreateSite creates a new site configuration
func CreateSite(site Site) error {
	sitesDir := GetSitesDir()
	if sitesDir == "" {
		return fmt.Errorf("nginx not installed")
	}

	// Create site root directory
	if site.Root != "" {
		os.MkdirAll(site.Root, 0755)
		// Create default index.php
		indexPath := filepath.Join(site.Root, "index.php")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			indexContent := fmt.Sprintf(`<?php
// Site: %s
// Created by VPS Panel
phpinfo();
`, site.Domain)
			os.WriteFile(indexPath, []byte(indexContent), 0644)
		}
	}

	// Generate config content
	config := generateSiteConfig(site)

	// Write config file
	configPath := filepath.Join(sitesDir, site.Name+".conf")
	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return err
	}

	// Update main nginx.conf to include sites
	if err := updateNginxMainConfig(); err != nil {
		return err
	}

	return nil
}

// generateSiteConfig generates nginx config for a site
func generateSiteConfig(site Site) string {
	// Ensure root path uses forward slashes for Nginx compatibility
	site.Root = strings.ReplaceAll(site.Root, "\\", "/")

	phpConfig := ""
	if site.PHPVersion != "" {
		phpCgiPath := GetPHPCGIPath(site.PHPVersion)
		phpCgiPath = strings.ReplaceAll(phpCgiPath, "\\", "/")

		phpConfig = fmt.Sprintf(`
    # PHP Version: %s
    location ~ \.php$ {
        fastcgi_pass   127.0.0.1:9000;
        fastcgi_index  index.php;
        fastcgi_param  SCRIPT_FILENAME  $document_root$fastcgi_script_name;
        include        fastcgi_params;
        # PHP-CGI: %s
    }`, site.PHPVersion, phpCgiPath)
	}

	sslConfig := ""
	if site.SSL {
		sslConfig = `
    ssl_certificate     ssl/server.crt;
    ssl_certificate_key ssl/server.key;`
	}

	listen := fmt.Sprintf("%d", site.Port)
	if site.SSL {
		listen += " ssl"
	}

	return fmt.Sprintf(`# Site: %s
# Created by VPS Panel

server {
    listen       %s;
    server_name  %s;

    root   %s;
    index  index.php index.html index.htm;
%s
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }
%s
    location ~ /\.ht {
        deny all;
    }

    error_page   500 502 503 504  /50x.html;
    location = /50x.html {
        root   html;
    }
}
`, site.Name, listen, site.Domain, site.Root, sslConfig, phpConfig)
}

// updateNginxMainConfig updates main nginx.conf to include sites directory
func updateNginxMainConfig() error {
	nginxPath := GetNginxPath()
	if nginxPath == "" {
		return fmt.Errorf("nginx not installed")
	}

	mainConfig := filepath.Join(nginxPath, "conf", "nginx.conf")
	content, err := os.ReadFile(mainConfig)
	if err != nil {
		return err
	}

	text := string(content)
	includeStatement := "include sites/*.conf;"

	// Check if already included
	if strings.Contains(text, includeStatement) {
		return nil
	}

	// Add include before the last closing brace of http block
	// Find the http block and add include
	if strings.Contains(text, "http {") {
		// Find last } and insert before it
		lastBrace := strings.LastIndex(text, "}")
		if lastBrace > 0 {
			text = text[:lastBrace] + "\n    " + includeStatement + "\n" + text[lastBrace:]
			return os.WriteFile(mainConfig, []byte(text), 0644)
		}
	}

	return nil
}

// DeleteSite deletes a site configuration
func DeleteSite(name string) error {
	sitesDir := GetSitesDir()
	if sitesDir == "" {
		return fmt.Errorf("nginx not installed")
	}

	configPath := filepath.Join(sitesDir, name+".conf")
	return os.Remove(configPath)
}

// GetSiteConfig returns the raw config content
func GetSiteConfig(name string) (string, error) {
	sitesDir := GetSitesDir()
	if sitesDir == "" {
		return "", fmt.Errorf("nginx not installed")
	}

	configPath := filepath.Join(sitesDir, name+".conf")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// SaveSiteConfig saves raw config content
func SaveSiteConfig(name, content string) error {
	sitesDir := GetSitesDir()
	if sitesDir == "" {
		return fmt.Errorf("nginx not installed")
	}

	configPath := filepath.Join(sitesDir, name+".conf")
	return os.WriteFile(configPath, []byte(content), 0644)
}

// GetNginxStatus returns nginx status
func GetNginxStatus() map[string]interface{} {
	nginxPath := GetNginxPath()

	status := map[string]interface{}{
		"installed":    nginxPath != "",
		"path":         nginxPath,
		"running":      false,
		"php_versions": GetInstalledPHPVersions(),
	}

	if nginxPath != "" {
		// Check if nginx is running
		svcStatus, err := appstore.GetServiceStatus("nginx", filepath.Base(nginxPath))
		if err == nil {
			status["running"] = svcStatus.Running
			status["version"] = svcStatus.Version
		}
	}

	return status
}
