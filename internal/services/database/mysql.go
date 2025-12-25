package database

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"vps-panel/internal/services/appstore"
)

// DatabaseInfo represents database information
type DatabaseInfo struct {
	Name   string `json:"name"`
	Tables int    `json:"tables"`
	Size   string `json:"size"`
}

// UserInfo represents database user
type UserInfo struct {
	User string `json:"user"`
	Host string `json:"host"`
}

// GetMySQLPath returns path to MySQL installation
func GetMySQLPath() string {
	baseDir := appstore.GetBaseDir()
	mysqlDir := filepath.Join(baseDir, "database", "mysql")

	entries, err := os.ReadDir(mysqlDir)
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if entry.IsDir() {
			return filepath.Join(mysqlDir, entry.Name())
		}
	}
	return ""
}

// GetMySQLVersion returns installed MySQL version
func GetMySQLVersion() string {
	mysqlPath := GetMySQLPath()
	if mysqlPath == "" {
		return ""
	}
	return filepath.Base(mysqlPath)
}

// IsMySQLRunning checks if MySQL is running
func IsMySQLRunning() bool {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("tasklist", "/FI", "IMAGENAME eq mysqld.exe")
	} else {
		cmd = exec.Command("pgrep", "mysqld")
	}

	output, err := cmd.Output()
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		return strings.Contains(string(output), "mysqld")
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// GetMySQLClient returns path to mysql client
func GetMySQLClient() string {
	mysqlPath := GetMySQLPath()
	if mysqlPath == "" {
		return ""
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(mysqlPath, "bin", "mysql.exe")
	}
	return filepath.Join(mysqlPath, "bin", "mysql")
}

// ExecuteQuery executes a MySQL query and returns results
func ExecuteQuery(query string) ([]map[string]interface{}, error) {
	client := GetMySQLClient()
	if client == "" {
		return nil, fmt.Errorf("MySQL client not found")
	}

	cmd := exec.Command(client, "-u", "root", "-e", query, "--batch", "--skip-column-names")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var results []map[string]interface{}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			results = append(results, map[string]interface{}{"result": line})
		}
	}

	return results, nil
}

// GetDatabases returns list of databases
func GetDatabases() ([]DatabaseInfo, error) {
	client := GetMySQLClient()
	if client == "" {
		return nil, fmt.Errorf("MySQL client not found")
	}

	cmd := exec.Command(client, "-u", "root", "-e", "SHOW DATABASES;", "--batch", "--skip-column-names")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get databases: %w", err)
	}

	var databases []DatabaseInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name != "" && name != "information_schema" && name != "performance_schema" && name != "sys" {
			db := DatabaseInfo{Name: name}
			// Get table count
			tableCmd := exec.Command(client, "-u", "root", "-e",
				fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='%s';", name),
				"--batch", "--skip-column-names")
			if tableOutput, err := tableCmd.Output(); err == nil {
				fmt.Sscanf(strings.TrimSpace(string(tableOutput)), "%d", &db.Tables)
			}
			databases = append(databases, db)
		}
	}

	return databases, nil
}

// CreateDatabase creates a new database
func CreateDatabase(name string) error {
	client := GetMySQLClient()
	if client == "" {
		return fmt.Errorf("MySQL client not found")
	}

	cmd := exec.Command(client, "-u", "root", "-e", fmt.Sprintf("CREATE DATABASE `%s`;", name))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create database: %s", string(output))
	}

	return nil
}

// DropDatabase drops a database
func DropDatabase(name string) error {
	client := GetMySQLClient()
	if client == "" {
		return fmt.Errorf("MySQL client not found")
	}

	// Safety check - don't drop system databases
	if name == "mysql" || name == "information_schema" || name == "performance_schema" || name == "sys" {
		return fmt.Errorf("cannot drop system database")
	}

	cmd := exec.Command(client, "-u", "root", "-e", fmt.Sprintf("DROP DATABASE `%s`;", name))
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to drop database: %s", string(output))
	}

	return nil
}

// GetUsers returns list of MySQL users
func GetUsers() ([]UserInfo, error) {
	client := GetMySQLClient()
	if client == "" {
		return nil, fmt.Errorf("MySQL client not found")
	}

	cmd := exec.Command(client, "-u", "root", "-e",
		"SELECT User, Host FROM mysql.user;", "--batch", "--skip-column-names")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	var users []UserInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			users = append(users, UserInfo{
				User: parts[0],
				Host: parts[1],
			})
		}
	}

	return users, nil
}

// CreateUser creates a new MySQL user
func CreateUser(username, password, host string) error {
	client := GetMySQLClient()
	if client == "" {
		return fmt.Errorf("MySQL client not found")
	}

	if host == "" {
		host = "localhost"
	}

	query := fmt.Sprintf("CREATE USER '%s'@'%s' IDENTIFIED BY '%s';", username, host, password)
	cmd := exec.Command(client, "-u", "root", "-e", query)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create user: %s", string(output))
	}

	return nil
}

// DropUser drops a MySQL user
func DropUser(username, host string) error {
	client := GetMySQLClient()
	if client == "" {
		return fmt.Errorf("MySQL client not found")
	}

	if username == "root" {
		return fmt.Errorf("cannot drop root user")
	}

	query := fmt.Sprintf("DROP USER '%s'@'%s';", username, host)
	cmd := exec.Command(client, "-u", "root", "-e", query)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to drop user: %s", string(output))
	}

	return nil
}

// GrantPrivileges grants all privileges on a database to a user
func GrantPrivileges(username, host, database string) error {
	client := GetMySQLClient()
	if client == "" {
		return fmt.Errorf("MySQL client not found")
	}

	query := fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'%s'; FLUSH PRIVILEGES;",
		database, username, host)
	cmd := exec.Command(client, "-u", "root", "-e", query)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to grant privileges: %s", string(output))
	}

	return nil
}

// GetStatus returns MySQL server status
func GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"installed": GetMySQLPath() != "",
		"running":   IsMySQLRunning(),
		"version":   GetMySQLVersion(),
		"path":      GetMySQLPath(),
	}

	if IsMySQLRunning() {
		client := GetMySQLClient()
		if client != "" {
			// Get uptime
			cmd := exec.Command(client, "-u", "root", "-e",
				"SHOW GLOBAL STATUS LIKE 'Uptime';", "--batch", "--skip-column-names")
			if output, err := cmd.Output(); err == nil {
				parts := strings.Split(strings.TrimSpace(string(output)), "\t")
				if len(parts) >= 2 {
					status["uptime"] = parts[1]
				}
			}
		}
	}

	return status
}
