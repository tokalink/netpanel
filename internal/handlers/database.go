package handlers

import (
	"path/filepath"
	"vps-panel/internal/services/appstore"
	dbservice "vps-panel/internal/services/database"

	"github.com/gofiber/fiber/v2"
)

// GetDatabaseStatus returns MySQL status
func GetDatabaseStatus(c *fiber.Ctx) error {
	status := dbservice.GetStatus()
	return c.JSON(status)
}

// GetDatabases returns list of databases
func GetDatabases(c *fiber.Ctx) error {
	databases, err := dbservice.GetDatabases()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(databases)
}

// CreateDatabase creates a new database
func CreateDatabase(c *fiber.Ctx) error {
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
			"error": "Database name is required",
		})
	}

	if err := dbservice.CreateDatabase(req.Name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Database created successfully",
	})
}

// DropDatabase drops a database
func DropDatabase(c *fiber.Ctx) error {
	name := c.Params("name")
	if name == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Database name is required",
		})
	}

	if err := dbservice.DropDatabase(name); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Database dropped",
	})
}

// GetDBUsers returns list of MySQL users
func GetDBUsers(c *fiber.Ctx) error {
	users, err := dbservice.GetUsers()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(users)
}

// CreateDBUser creates a new MySQL user
func CreateDBUser(c *fiber.Ctx) error {
	type Request struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Host     string `json:"host"`
		Database string `json:"database"`
	}

	var req Request
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username and password are required",
		})
	}

	if req.Host == "" {
		req.Host = "localhost"
	}

	if err := dbservice.CreateUser(req.Username, req.Password, req.Host); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Grant privileges if database specified
	if req.Database != "" {
		if err := dbservice.GrantPrivileges(req.Username, req.Host, req.Database); err != nil {
			return c.Status(500).JSON(fiber.Map{
				"error": "User created but failed to grant privileges: " + err.Error(),
			})
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User created successfully",
	})
}

// DropDBUser drops a MySQL user
func DropDBUser(c *fiber.Ctx) error {
	username := c.Params("username")
	host := c.Query("host", "localhost")

	if username == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "Username is required",
		})
	}

	if err := dbservice.DropUser(username, host); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "User dropped",
	})
}

// StartMySQL starts MySQL service
func StartMySQL(c *fiber.Ctx) error {
	mysqlPath := dbservice.GetMySQLPath()
	if mysqlPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "MySQL not installed",
		})
	}

	version := filepath.Base(mysqlPath)
	if err := appstore.StartService("mysql", version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "MySQL started",
	})
}

// StopMySQL stops MySQL service
func StopMySQL(c *fiber.Ctx) error {
	mysqlPath := dbservice.GetMySQLPath()
	if mysqlPath == "" {
		return c.Status(404).JSON(fiber.Map{
			"error": "MySQL not installed",
		})
	}

	version := filepath.Base(mysqlPath)
	if err := appstore.StopService("mysql", version); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "MySQL stopped",
	})
}
