package appstore

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ServiceStatus represents the status of a service
type ServiceStatus struct {
	PackageID   string `json:"package_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Running     bool   `json:"running"`
	PID         int    `json:"pid,omitempty"`
	Port        int    `json:"port,omitempty"`
	InstallPath string `json:"install_path"`
	ConfigPath  string `json:"config_path,omitempty"`
	LogPath     string `json:"log_path,omitempty"`
}

// GetServiceStatus checks if a service is running
func GetServiceStatus(packageID, version string) (*ServiceStatus, error) {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)
	if _, err := os.Stat(installPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("package not installed: %s %s", packageID, version)
	}

	status := &ServiceStatus{
		PackageID:   packageID,
		Name:        pkg.Name,
		Version:     version,
		InstallPath: installPath,
		Running:     false,
	}

	// Set ports
	if len(pkg.Ports) > 0 {
		status.Port = pkg.Ports[0]
	}

	// Set config path
	if pkg.ConfigFile != "" {
		status.ConfigPath = filepath.Join(installPath, pkg.ConfigFile)
	}

	// Check if process is running based on package type
	switch packageID {
	case "nginx":
		status.Running = isProcessRunning("nginx")
		status.ConfigPath = filepath.Join(installPath, "conf", "nginx.conf")
		status.LogPath = filepath.Join(installPath, "logs")
	case "mysql", "mariadb":
		status.Running = isProcessRunning("mysqld") || isProcessRunning("mariadbd")
		status.ConfigPath = filepath.Join(installPath, "my.ini")
		status.LogPath = filepath.Join(installPath, "data")
	case "redis":
		status.Running = isProcessRunning("redis-server")
		status.ConfigPath = filepath.Join(installPath, "redis.conf")
	case "php":
		// PHP is not a service, just check if executable exists
		execPath := filepath.Join(installPath, pkg.Executable[runtime.GOOS])
		if _, err := os.Stat(execPath); err == nil {
			status.Running = true // Available
		}
	case "nodejs":
		// Node.js is not a service
		execPath := filepath.Join(installPath, pkg.Executable[runtime.GOOS])
		if _, err := os.Stat(execPath); err == nil {
			status.Running = true
		}
	}

	return status, nil
}

// isProcessRunning checks if a process with given name is running
func isProcessRunning(processName string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("tasklist", "/FI", fmt.Sprintf("IMAGENAME eq %s*", processName))
	default:
		cmd = exec.Command("pgrep", "-x", processName)
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return strings.Contains(string(output), processName)
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// StartService starts a service
func StartService(packageID, version string) error {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)
	execName := pkg.Executable[runtime.GOOS]
	if execName == "" {
		return fmt.Errorf("no executable defined for %s on %s", packageID, runtime.GOOS)
	}

	execPath := filepath.Join(installPath, execName)
	if _, err := os.Stat(execPath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s", execPath)
	}

	var cmd *exec.Cmd

	switch packageID {
	case "nginx":
		// Nginx: start with -p for prefix path
		if runtime.GOOS == "windows" {
			cmd = exec.Command(execPath, "-p", installPath)
		} else {
			cmd = exec.Command(execPath, "-p", installPath, "-c", filepath.Join(installPath, "conf", "nginx.conf"))
		}
	case "mysql", "mariadb":
		// MySQL/MariaDB
		dataDir := filepath.Join(installPath, "data")
		os.MkdirAll(dataDir, 0755)

		// Create my.ini config if not exists
		configFile := filepath.Join(installPath, "my.ini")
		if runtime.GOOS != "windows" {
			configFile = filepath.Join(installPath, "my.cnf")
		}

		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			configContent := fmt.Sprintf("[mysqld]\nport=3306\nbasedir=%s\ndatadir=%s\n", installPath, dataDir)
			os.WriteFile(configFile, []byte(configContent), 0644)
		}

		// Get mysqld path (add .exe on Windows)
		mysqldPath := filepath.Join(installPath, "bin", "mysqld")
		if runtime.GOOS == "windows" {
			mysqldPath += ".exe"
		}

		// Initialize data dir if ibdata1 doesn't exist (better check than mysql folder)
		ibdataFile := filepath.Join(dataDir, "ibdata1")
		if _, err := os.Stat(ibdataFile); os.IsNotExist(err) {
			// Clear data dir first
			os.RemoveAll(dataDir)
			os.MkdirAll(dataDir, 0755)

			initCmd := exec.Command(mysqldPath,
				"--initialize-insecure",
				"--basedir="+installPath,
				"--datadir="+dataDir,
				"--console")
			initCmd.Dir = installPath
			initCmd.Run() // Wait for init to complete
		}

		cmd = exec.Command(mysqldPath,
			"--basedir="+installPath,
			"--datadir="+dataDir,
			"--port=3306",
			"--console")
	case "redis":
		configFile := filepath.Join(installPath, "redis.conf")
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			// Create default config
			os.WriteFile(configFile, []byte("bind 127.0.0.1\nport 6379\n"), 0644)
		}
		cmd = exec.Command(execPath, configFile)
	default:
		cmd = exec.Command(execPath)
	}

	cmd.Dir = installPath

	// Start in background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	return nil
}

// StopService stops a running service
func StopService(packageID, version string) error {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)

	switch packageID {
	case "nginx":
		execPath := filepath.Join(installPath, pkg.Executable[runtime.GOOS])
		cmd := exec.Command(execPath, "-s", "stop", "-p", installPath)
		cmd.Dir = installPath
		err := cmd.Run()
		if err != nil {
			// Fallback to taskkill
			killProcess("nginx")
		}
		return nil
	case "mysql", "mariadb":
		// Use mysqladmin shutdown
		adminPath := filepath.Join(installPath, "bin", "mysqladmin")
		if runtime.GOOS == "windows" {
			adminPath += ".exe"
		}
		cmd := exec.Command(adminPath, "-u", "root", "shutdown")
		err := cmd.Run()
		if err != nil {
			// Fallback to taskkill
			killProcess("mysqld")
		}
		return nil
	case "redis":
		// Use redis-cli shutdown
		cliPath := filepath.Join(installPath, "redis-cli")
		if runtime.GOOS == "windows" {
			cliPath += ".exe"
		} else {
			cliPath = filepath.Join(installPath, "src", "redis-cli")
		}
		cmd := exec.Command(cliPath, "shutdown")
		err := cmd.Run()
		if err != nil {
			killProcess("redis-server")
		}
		return nil
	case "php", "nodejs", "phpmyadmin", "adminer", "composer":
		// These are not services, nothing to stop
		return nil
	default:
		// Generic process kill
		return killProcess(packageID)
	}
}

// RestartService restarts a service
func RestartService(packageID, version string) error {
	StopService(packageID, version)
	return StartService(packageID, version)
}

// killProcess kills a process by name
func killProcess(processName string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", processName+"*")
	default:
		cmd = exec.Command("pkill", "-9", processName)
	}

	return cmd.Run()
}

// GetConfig reads configuration file content
func GetConfig(packageID, version string) (string, string, error) {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return "", "", fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)

	var configPath string
	switch packageID {
	case "nginx":
		configPath = filepath.Join(installPath, "conf", "nginx.conf")
	case "mysql", "mariadb":
		if runtime.GOOS == "windows" {
			configPath = filepath.Join(installPath, "my.ini")
		} else {
			configPath = filepath.Join(installPath, "my.cnf")
		}
	case "redis":
		configPath = filepath.Join(installPath, "redis.conf")
	case "php":
		configPath = filepath.Join(installPath, "php.ini")
	default:
		if pkg.ConfigFile != "" {
			configPath = filepath.Join(installPath, pkg.ConfigFile)
		} else {
			return "", "", fmt.Errorf("no config file for %s", packageID)
		}
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		// Return default config if file doesn't exist
		defaultConfig := getDefaultConfig(packageID, installPath)
		return configPath, defaultConfig, nil
	}

	return configPath, string(content), nil
}

// SaveConfig saves configuration file content
func SaveConfig(packageID, version, content string) error {
	pkg := GetPortablePackageByID(packageID)
	if pkg == nil {
		return fmt.Errorf("package not found: %s", packageID)
	}

	installPath := filepath.Join(GetBaseDir(), pkg.InstallPath, version)

	var configPath string
	switch packageID {
	case "nginx":
		configPath = filepath.Join(installPath, "conf", "nginx.conf")
	case "mysql", "mariadb":
		if runtime.GOOS == "windows" {
			configPath = filepath.Join(installPath, "my.ini")
		} else {
			configPath = filepath.Join(installPath, "my.cnf")
		}
	case "redis":
		configPath = filepath.Join(installPath, "redis.conf")
	case "php":
		configPath = filepath.Join(installPath, "php.ini")
	default:
		if pkg.ConfigFile != "" {
			configPath = filepath.Join(installPath, pkg.ConfigFile)
		} else {
			return fmt.Errorf("no config file for %s", packageID)
		}
	}

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(configPath), 0755)

	return os.WriteFile(configPath, []byte(content), 0644)
}

// getDefaultConfig returns default configuration content
func getDefaultConfig(packageID, installPath string) string {
	switch packageID {
	case "nginx":
		return fmt.Sprintf(`worker_processes 1;

events {
    worker_connections 1024;
}

http {
    include       mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout 65;

    server {
        listen       80;
        server_name  localhost;

        root   %s/html;
        index  index.html index.htm index.php;

        location / {
            try_files $uri $uri/ =404;
        }

        location ~ \.php$ {
            fastcgi_pass   127.0.0.1:9000;
            fastcgi_index  index.php;
            fastcgi_param  SCRIPT_FILENAME  $document_root$fastcgi_script_name;
            include        fastcgi_params;
        }
    }
}
`, installPath)
	case "mysql", "mariadb":
		return fmt.Sprintf(`[mysqld]
port=3306
basedir=%s
datadir=%s/data
socket=%s/mysql.sock
log-error=%s/error.log
pid-file=%s/mysql.pid

[client]
port=3306
socket=%s/mysql.sock
`, installPath, installPath, installPath, installPath, installPath, installPath)
	case "redis":
		return `bind 127.0.0.1
port 6379
daemonize no
loglevel notice
databases 16
save 900 1
save 300 10
save 60 10000
`
	case "php":
		return `[PHP]
engine = On
short_open_tag = Off
precision = 14
output_buffering = 4096
zlib.output_compression = Off
implicit_flush = Off
serialize_precision = -1
disable_functions =
disable_classes =
zend.enable_gc = On
expose_php = Off
max_execution_time = 30
max_input_time = 60
memory_limit = 256M
error_reporting = E_ALL
display_errors = Off
display_startup_errors = Off
log_errors = On
error_log = php_errors.log
post_max_size = 128M
upload_max_filesize = 128M
max_file_uploads = 20
date.timezone = Asia/Jakarta

[Session]
session.save_handler = files
session.use_strict_mode = 1
session.use_cookies = 1
session.use_only_cookies = 1
session.name = PHPSESSID
session.auto_start = 0
session.cookie_lifetime = 0
session.gc_maxlifetime = 1440
`
	default:
		return ""
	}
}
