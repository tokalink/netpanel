# VPS Panel - Development README

## Summary
Complete VPS Panel web application with dashboard, app store, services management, web server configuration, database management, file manager, and Docker container support.

## Features Implemented

### 1. Dashboard
- Real-time system metrics (CPU, RAM, Disk, Network)
- WebSocket-powered live updates
- Quick stats overview

### 2. App Store
- **Table-based UI** with sortable columns
- **Multi-version support** - Each version displayed as separate row
- **Install multiple PHP versions** (8.4.16, 8.3.29, 8.2.30, 8.1.34)
- **Progress modal** with:
  - Progress bar with percentage
  - Status text
  - Color-coded log output (info/success/error)
- **Uninstall feature** with confirmation
- Category filtering (Database, Runtime, Web Server, Tools)
- Search functionality

### 3. Services
- List all installed services with status
- Start/Stop/Restart controls
- View service logs

### 4. Web Server (Nginx)
- Nginx status monitoring
- Site management (add/edit/delete)
- PHP-CGI integration with version selector
- Start/Stop PHP-CGI controls
- Configuration per site (document root, PHP version)

### 5. Database (MySQL)
- MySQL status and version display
- Start/Stop MySQL
- **Databases tab:**
  - List all databases with table count
  - Create new database
  - Drop database
- **Users tab:**
  - List all users
  - Create user with password
  - Grant privileges to database
  - Drop user

### 6. File Manager
- Browse files in `/server/` directory
- Breadcrumb navigation
- Double-click to open folder or edit file
- Right-click context menu (Rename, Download, Delete)
- Create new folder/file
- Upload files (drag & drop support)
- Built-in code editor with save

### 7. Docker Containers
- Docker status check
- **Containers tab:**
  - List all containers (running & stopped)
  - Start/Stop/Restart containers
  - View container logs
  - Remove containers
- **Images tab:**
  - List all images
  - Pull new images
  - Run container from image
  - Remove images

### 8. System Tools
- **Cron Jobs:**
  - Internal cron scheduler (robfig/cron)
  - Manage scheduled tasks (Add/Delete/Toggle)
  - Support for standard cron syntax
  - View status and last execution result
- **Firewall (Windows):**
  - Manage Windows Firewall rules via `netsh`
  - List managed rules
  - Add Allow/Block rules for TCP/UDP ports
  - Delete rules
- **Web Terminal:**
  - Fully functional web-based terminal
  - **Shell Support**: Windows (CMD/PowerShell) and Linux (Bash).
  - **Responsive**: Adapts to browser window size.
  - Real-time interaction via WebSocket & PTY
- **Kubernetes (K8s):**
  - Menu item added (Feature coming soon)

## Production Build

To creating a production-ready release, use the provided build scripts:

### Windows (PowerShell)
```powershell
.\build.ps1
```

### Linux / macOS (Bash)
```bash
./build-linux.sh
```

These scripts will create a `build/` folder containing:
- Executable binaries for all platforms (Windows, Linux, macOS).
- The `web/` directory (templates & static files).
- The `config.yaml` file.
- The `data/` directory.

To deploy, simply copy the contents of the `build/` folder to your server.

## Technical Details

### Backend (Go/Fiber)
- 131 API handlers
- JWT authentication
- SQLite database for state
- WebSocket for real-time stats

### Frontend
- Vanilla HTML/CSS/JavaScript
- Responsive design
- Dark sidebar navigation
- Modal-based forms

## API Endpoints

### App Store
- `GET /api/portable/packages` - List available packages
- `GET /api/portable/installed` - List installed packages
- `POST /api/portable/install` - Install package
- `DELETE /api/portable/packages/:id` - Uninstall package

### Services
- `GET /api/service/:id/status` - Get service status
- `POST /api/service/:id/start` - Start service
- `POST /api/service/:id/stop` - Stop service

### Web Server
- `GET /api/webserver/status` - Nginx status
- `GET /api/webserver/sites` - List sites
- `POST /api/webserver/sites` - Create site
- `DELETE /api/webserver/sites/:name` - Delete site

### Database
- `GET /api/database/status` - MySQL status
- `GET /api/database/databases` - List databases
- `POST /api/database/databases` - Create database
- `DELETE /api/database/databases/:name` - Drop database
- `GET /api/database/users` - List users
- `POST /api/database/users` - Create user

### File Manager
- `GET /api/files/list` - List files
- `GET /api/files/read` - Read file content
- `POST /api/files/save` - Save file
- `POST /api/files/upload` - Upload file
- `DELETE /api/files/delete` - Delete file/folder

### Docker
- `GET /api/docker/status` - Docker status
- `GET /api/docker/containers` - List containers
- `POST /api/docker/containers/:id/start` - Start container
- `POST /api/docker/containers/:id/stop` - Stop container
- `GET /api/docker/images` - List images
- `POST /api/docker/images/pull` - Pull image

### System Tools
- `GET /api/cron/jobs` - List cron jobs
- `POST /api/cron/jobs` - Add cron job
- `Delete /api/cron/jobs/:id` - Delete cron job
- `POST /api/cron/jobs/:id/toggle` - Enable/disable job
- `GET /api/firewall/rules` - List firewall rules
- `POST /api/firewall/rules` - Add firewall rule
- `DELETE /api/firewall/rules` - Delete firewall rule

## Running the Application
```bash
cd d:\go\vps-panel
go build -o vps-panel.exe ./cmd/server
.\vps-panel.exe
```

Open http://localhost:8989 in browser.

Default login: `admin` / `admin123`
