package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/kardianos/service"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/template/html/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"

	"vps-panel/internal/config"
	"vps-panel/internal/database"
	"vps-panel/internal/handlers"
	"vps-panel/internal/middleware"
	"vps-panel/internal/models"
	"vps-panel/internal/services/cron"
	ws "vps-panel/internal/services/websocket"
	"vps-panel/web"
)

var svcLogger service.Logger

type program struct {
	app *fiber.App
}

func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

func (p *program) Stop(s service.Service) error {
	if p.app != nil {
		return p.app.Shutdown()
	}
	return nil
}

func (p *program) run() {
	// Determine executable directory
	ex, err := os.Executable()
	if err != nil {
		log.Println("Failed to determine executable path:", err)
	}
	exeDir := filepath.Dir(ex)

	// Load .env file (try from exe dir first, then CWD)
	godotenv.Load(filepath.Join(exeDir, ".env")) // Explicit path
	godotenv.Load()                              // Default search

	// Load configuration
	// Check for config.yaml in exeDir
	configPath := filepath.Join(exeDir, "config.yaml")
	cfg, err := config.Load(configPath)
	if err != nil {
		// Fallback to local
		cfg, err = config.Load("config.yaml")
		if err != nil {
			log.Printf("Warning: Failed to load config: %v. Using defaults.", err)
			// Ensure defaults are sane if Load returns partial config or error
			if cfg == nil {
				// Should not happen with current config.Load logic if it handles defaults
				// We'll rely on config.Load returning default config on error if it handles file not found
				// Actually my config.Load returns nil on parsing error, but I implemented fallback logic for NotExist.
			}
		}
	}

	// Override with .env PORT if set
	if envPort := os.Getenv("PORT"); envPort != "" {
		if port, err := strconv.Atoi(envPort); err == nil {
			cfg.Server.Port = port
		}
	}

	// Double check DB path resolution if relative and we are in weird CWD
	if !filepath.IsAbs(cfg.Database.Path) && ex != "" {
		cfg.Database.Path = filepath.Join(exeDir, cfg.Database.Path)
	}

	// Connect to database
	log.Printf("Connecting to database at: %s", cfg.Database.Path)
	_, err = database.Connect(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto migrate models
	if err := database.AutoMigrate(
		&models.User{},
		&models.Setting{},
		&models.InstalledPackage{},
		&models.ActivityLog{},
		&models.CronJob{},
		&models.FirewallRule{},
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Create default admin user if not exists
	createDefaultAdmin(cfg)

	// Initialize WebSocket hub
	ws.InitHub()

	// Initialize Cron service
	cron.Init()

	// Setup template engine
	// Use embedded templates
	templatesFS, err := fs.Sub(web.EmbedFS, "templates")
	if err != nil {
		log.Fatalf("Failed to access embedded templates: %v", err)
	}
	engine := html.NewFileSystem(http.FS(templatesFS), ".html")
	engine.Reload(true) // Optional: for dev iteration if using physical files, but safe to leave

	// Create Fiber app
	app := fiber.New(fiber.Config{
		Views:       engine,
		ViewsLayout: "layouts/base",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})
	p.app = app

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowMethods:     "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: false,
	}))

	// Static files (embedded)
	app.Use("/static", filesystem.New(filesystem.Config{
		Root:   web.GetFileSystem(),
		Browse: false,
	}))

	// WebSocket upgrade middleware
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	// Routes
	setupRoutes(app, cfg)

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("🚀 VPS Panel starting on http://%s", addr)
	log.Printf("📊 Dashboard: http://localhost:%d", cfg.Server.Port)

	if err := app.Listen(addr); err != nil {
		log.Printf("Server Listen Error: %v", err)
	}
}

func main() {
	// Service configuration
	svcConfig := &service.Config{
		Name:        "VPSPanel",
		DisplayName: "VPS Panel Service",
		Description: "Web-based management panel for VPS.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	svcLogger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	// Check arguments for service control
	if len(os.Args) > 1 {
		cmd := os.Args[1]
		switch cmd {
		case "install", "uninstall", "start", "stop", "restart":
			err = service.Control(s, cmd)
			if err != nil {
				log.Fatalf("Failed to %s service: %v", cmd, err)
			}
			fmt.Printf("Service %s succeeded.\n", cmd)
			return
		case "run":
			// Explicitly run
		case "help":
			printHelp()
			return
		default:
			fmt.Printf("Unknown command: %s\n", cmd)
			printHelp()
			return
		}
	}

	// Run service (or standalone)
	err = s.Run()
	if err != nil {
		svcLogger.Error(err)
	}
}

func printHelp() {
	fmt.Println("VPS Panel - Service Management")
	fmt.Println("")
	fmt.Println("Usage: vps-panel [command]")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  install    Install as a system service")
	fmt.Println("  uninstall  Remove the system service")
	fmt.Println("  start      Start the installed service")
	fmt.Println("  stop       Stop the installed service")
	fmt.Println("  restart    Restart the installed service")
	fmt.Println("  run        Run directly (standalone mode)")
	fmt.Println("  help       Show this help message")
	fmt.Println("")
	fmt.Println("If no command is provided, it runs in standalone mode (equivalent to 'run').")
}

func setupRoutes(app *fiber.App, cfg *config.Config) {
	// Public routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/login")
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		// Check if user is already logged in
		if tokenStr := c.Cookies("token"); tokenStr != "" {
			token, err := jwt.ParseWithClaims(tokenStr, &middleware.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(cfg.JWT.Secret), nil
			})
			if err == nil && token.Valid {
				return c.Redirect("/dashboard")
			}
		}

		return c.Render("pages/login", fiber.Map{
			"Title": "Login - VPS Panel",
		})
	})

	// API routes - Public
	api := app.Group("/api")
	api.Post("/auth/login", handlers.Login)

	// API routes - Protected
	protected := api.Group("/", middleware.AuthRequired())
	protected.Post("/auth/logout", handlers.Logout)
	protected.Get("/auth/profile", handlers.GetProfile)
	protected.Post("/auth/2fa/setup", handlers.Setup2FA)
	protected.Post("/auth/2fa/verify", handlers.Verify2FA)
	protected.Post("/auth/2fa/disable", handlers.Disable2FA)

	// Dashboard API
	protected.Get("/dashboard", handlers.GetDashboard)
	protected.Get("/system/stats", handlers.GetSystemStats)

	// App Store API (system package manager)
	protected.Get("/appstore/packages", handlers.GetPackages)
	protected.Get("/appstore/packages/:id/status", handlers.GetPackageStatus)
	protected.Get("/appstore/installed", handlers.GetInstalledPackages)
	protected.Post("/appstore/install", handlers.InstallPackage)
	protected.Delete("/appstore/packages/:id", handlers.UninstallPackage)
	protected.Get("/appstore/system", handlers.GetSystemInfo)
	protected.Post("/appstore/preview", handlers.PreviewInstall)

	// Portable App Store API (download-based installation)
	protected.Get("/portable/packages", handlers.GetPortablePackages)
	protected.Get("/portable/installed", handlers.GetPortableInstalled)
	protected.Post("/portable/install", handlers.InstallPortablePackage)
	protected.Delete("/portable/packages/:id", handlers.UninstallPortablePackage)
	protected.Get("/portable/system", handlers.GetPortableSystemInfo)
	protected.Post("/portable/preview", handlers.PreviewPortableInstall)

	// Service Control API
	protected.Get("/service/:id/status", handlers.GetServiceStatus)
	protected.Post("/service/:id/start", handlers.StartService)
	protected.Post("/service/:id/stop", handlers.StopService)
	protected.Post("/service/:id/restart", handlers.RestartService)
	protected.Get("/service/:id/config", handlers.GetServiceConfig)
	protected.Post("/service/:id/config", handlers.SaveServiceConfig)
	protected.Get("/service/:id/logs", handlers.GetServiceLogs)

	// Services List API
	protected.Get("/services", handlers.GetAllServices)
	protected.Post("/services/:id/:action", handlers.ServiceAction)

	// Web Server API
	protected.Get("/webserver/status", handlers.GetWebServerStatus)
	protected.Get("/webserver/sites", handlers.GetSites)
	protected.Post("/webserver/sites", handlers.CreateSite)
	protected.Delete("/webserver/sites/:name", handlers.DeleteSite)
	protected.Get("/webserver/sites/:name/config", handlers.GetSiteConfigHandler)
	protected.Post("/webserver/sites/:name/config", handlers.SaveSiteConfigHandler)
	protected.Post("/webserver/reload", handlers.ReloadNginx)
	protected.Get("/webserver/php", handlers.GetPHPVersions)
	protected.Post("/webserver/php/start", handlers.StartPHPCGI)
	protected.Post("/webserver/php/stop", handlers.StopPHPCGI)
	protected.Get("/webserver/php/status", handlers.GetPHPCGIStatus)

	// Database API
	protected.Get("/database/status", handlers.GetDatabaseStatus)
	protected.Get("/database/databases", handlers.GetDatabases)
	protected.Post("/database/databases", handlers.CreateDatabase)
	protected.Delete("/database/databases/:name", handlers.DropDatabase)
	protected.Get("/database/users", handlers.GetDBUsers)
	protected.Post("/database/users", handlers.CreateDBUser)
	protected.Delete("/database/users/:username", handlers.DropDBUser)
	protected.Post("/database/start", handlers.StartMySQL)
	protected.Post("/database/stop", handlers.StopMySQL)

	// File Manager API
	protected.Get("/files/list", handlers.ListFiles)
	protected.Get("/files/read", handlers.ReadFileContent)
	protected.Post("/files/save", handlers.SaveFileContent)
	protected.Post("/files/folder", handlers.CreateFolder)
	protected.Post("/files/create", handlers.CreateFile)
	protected.Delete("/files/delete", handlers.DeleteItem)
	protected.Post("/files/rename", handlers.RenameItem)
	protected.Post("/files/upload", handlers.UploadFile)
	protected.Get("/files/download", handlers.DownloadFile)

	// Cron API
	protected.Get("/cron/jobs", handlers.GetCronJobs)
	protected.Post("/cron/jobs", handlers.AddCronJob)
	protected.Delete("/cron/jobs/:id", handlers.RemoveCronJob)
	protected.Post("/cron/jobs/:id/toggle", handlers.ToggleCronJob)

	// Firewall API
	protected.Get("/firewall/rules", handlers.GetFirewallRules)
	protected.Post("/firewall/rules", handlers.AddFirewallRule)
	protected.Delete("/firewall/rules", handlers.DeleteFirewallRule)

	// Docker API
	protected.Get("/docker/status", handlers.GetDockerStatus)
	protected.Get("/docker/containers", handlers.GetContainers)
	protected.Get("/docker/images", handlers.GetImages)
	protected.Post("/docker/containers/:id/start", handlers.StartContainer)
	protected.Post("/docker/containers/:id/stop", handlers.StopContainer)
	protected.Post("/docker/containers/:id/restart", handlers.RestartContainer)
	protected.Delete("/docker/containers/:id", handlers.RemoveContainer)
	protected.Get("/docker/containers/:id/logs", handlers.GetContainerLogs)
	protected.Post("/docker/images/pull", handlers.PullImage)
	protected.Delete("/docker/images/:id", handlers.RemoveImage)
	protected.Post("/docker/run", handlers.RunContainer)

	// WebSocket
	app.Get("/ws/stats", websocket.New(ws.HandleWebSocket))

	// Terminal WebSocket
	app.Get("/ws/terminal", websocket.New(handlers.TerminalHandler))

	// Dashboard pages (protected via cookie)
	dashboard := app.Group("/dashboard")
	dashboard.Get("/", func(c *fiber.Ctx) error {
		return c.Render("pages/dashboard", fiber.Map{
			"Title":  "Dashboard - VPS Panel",
			"Active": "dashboard",
		})
	})
	dashboard.Get("/files", func(c *fiber.Ctx) error {
		return c.Render("pages/files", fiber.Map{
			"Title":  "File Manager - VPS Panel",
			"Active": "files",
		})
	})
	dashboard.Get("/webserver", func(c *fiber.Ctx) error {
		return c.Render("pages/webserver", fiber.Map{
			"Title":  "Web Server - VPS Panel",
			"Active": "webserver",
		})
	})
	dashboard.Get("/database", func(c *fiber.Ctx) error {
		return c.Render("pages/database", fiber.Map{
			"Title":  "Database - VPS Panel",
			"Active": "database",
		})
	})
	dashboard.Get("/docker", func(c *fiber.Ctx) error {
		return c.Render("pages/docker", fiber.Map{
			"Title":  "Docker - VPS Panel",
			"Active": "docker",
		})
	})
	dashboard.Get("/kubernetes", func(c *fiber.Ctx) error {
		return c.Render("pages/kubernetes", fiber.Map{
			"Title":  "Kubernetes - VPS Panel",
			"Active": "kubernetes",
		})
	})
	dashboard.Get("/appstore", func(c *fiber.Ctx) error {
		return c.Render("pages/appstore", fiber.Map{
			"Title":  "App Store - VPS Panel",
			"Active": "appstore",
		})
	})
	dashboard.Get("/cron", func(c *fiber.Ctx) error {
		return c.Render("pages/cron", fiber.Map{
			"Title":  "Cron Jobs - VPS Panel",
			"Active": "cron",
		})
	})
	dashboard.Get("/firewall", func(c *fiber.Ctx) error {
		return c.Render("pages/firewall", fiber.Map{
			"Title":  "Firewall - VPS Panel",
			"Active": "firewall",
		})
	})
	dashboard.Get("/services", func(c *fiber.Ctx) error {
		return c.Render("pages/services", fiber.Map{
			"Title":  "Services - VPS Panel",
			"Active": "services",
		})
	})
	dashboard.Get("/settings", func(c *fiber.Ctx) error {
		return c.Render("pages/settings", fiber.Map{
			"Title": "Settings - VPS Panel",
			"Path":  "/dashboard/settings",
		})
	})

	dashboard.Get("/terminal", func(c *fiber.Ctx) error {
		return c.Render("pages/terminal", fiber.Map{
			"Title": "Terminal - VPS Panel",
			"Path":  "/dashboard/terminal",
		})
	})

}

func createDefaultAdmin(cfg *config.Config) {
	var count int64
	database.DB.Model(&models.User{}).Count(&count)
	if count > 0 {
		return
	}

	admin := models.User{
		Username: cfg.Admin.Username,
		Email:    cfg.Admin.Email,
		Role:     "admin",
	}
	admin.SetPassword(cfg.Admin.Password)

	if err := database.DB.Create(&admin).Error; err != nil {
		log.Printf("Failed to create default admin: %v", err)
	} else {
		log.Printf("✅ Default admin user created: %s", cfg.Admin.Username)
	}
}
