package handlers

import (
	"encoding/json"
	"os/exec"
	"runtime"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// Container represents a Docker container
type Container struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Image   string `json:"image"`
	Status  string `json:"status"`
	State   string `json:"state"`
	Ports   string `json:"ports"`
	Created string `json:"created"`
}

// Image represents a Docker image
type Image struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Size       string `json:"size"`
	Created    string `json:"created"`
}

// isDockerInstalled checks if Docker is available
func isDockerInstalled() bool {
	cmd := exec.Command("docker", "version")
	return cmd.Run() == nil
}

// GetDockerStatus returns Docker status
func GetDockerStatus(c *fiber.Ctx) error {
	installed := isDockerInstalled()

	status := fiber.Map{
		"installed": installed,
		"running":   false,
		"version":   "",
	}

	if installed {
		// Get Docker version
		cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
		if output, err := cmd.Output(); err == nil {
			status["version"] = strings.TrimSpace(string(output))
			status["running"] = true
		}

		// Get container count
		cmd = exec.Command("docker", "ps", "-q")
		if output, err := cmd.Output(); err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if lines[0] != "" {
				status["running_containers"] = len(lines)
			} else {
				status["running_containers"] = 0
			}
		}
	}

	return c.JSON(status)
}

// GetContainers returns list of all containers
func GetContainers(c *fiber.Ctx) error {
	if !isDockerInstalled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "Docker not installed",
		})
	}

	// Get all containers (including stopped)
	cmd := exec.Command("docker", "ps", "-a", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list containers",
		})
	}

	var containers []Container
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		container := Container{
			ID:      getString(raw, "ID"),
			Name:    strings.TrimPrefix(getString(raw, "Names"), "/"),
			Image:   getString(raw, "Image"),
			Status:  getString(raw, "Status"),
			State:   getString(raw, "State"),
			Ports:   getString(raw, "Ports"),
			Created: getString(raw, "CreatedAt"),
		}
		containers = append(containers, container)
	}

	return c.JSON(containers)
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetImages returns list of Docker images
func GetImages(c *fiber.Ctx) error {
	if !isDockerInstalled() {
		return c.Status(503).JSON(fiber.Map{
			"error": "Docker not installed",
		})
	}

	cmd := exec.Command("docker", "images", "--format", "{{json .}}")
	output, err := cmd.Output()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to list images",
		})
	}

	var images []Image
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue
		}

		image := Image{
			ID:         getString(raw, "ID"),
			Repository: getString(raw, "Repository"),
			Tag:        getString(raw, "Tag"),
			Size:       getString(raw, "Size"),
			Created:    getString(raw, "CreatedAt"),
		}
		images = append(images, image)
	}

	return c.JSON(images)
}

// StartContainer starts a container
func StartContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Container ID required"})
	}

	cmd := exec.Command("docker", "start", id)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": string(output),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Container started"})
}

// StopContainer stops a container
func StopContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Container ID required"})
	}

	cmd := exec.Command("docker", "stop", id)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": string(output),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Container stopped"})
}

// RestartContainer restarts a container
func RestartContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Container ID required"})
	}

	cmd := exec.Command("docker", "restart", id)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": string(output),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Container restarted"})
}

// RemoveContainer removes a container
func RemoveContainer(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Container ID required"})
	}

	// Force remove
	cmd := exec.Command("docker", "rm", "-f", id)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": string(output),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Container removed"})
}

// GetContainerLogs returns container logs
func GetContainerLogs(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Container ID required"})
	}

	// Get last 100 lines
	cmd := exec.Command("docker", "logs", "--tail", "100", id)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "Failed to get logs",
		})
	}

	return c.JSON(fiber.Map{
		"logs": string(output),
	})
}

// PullImage pulls a Docker image
func PullImage(c *fiber.Ctx) error {
	type PullRequest struct {
		Image string `json:"image"`
	}

	var req PullRequest
	if err := c.BodyParser(&req); err != nil || req.Image == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Image name required"})
	}

	cmd := exec.Command("docker", "pull", req.Image)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Failed to pull image",
			"output": string(output),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Image pulled",
		"output":  string(output),
	})
}

// RemoveImage removes a Docker image
func RemoveImage(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Image ID required"})
	}

	cmd := exec.Command("docker", "rmi", id)
	if output, err := cmd.CombinedOutput(); err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": string(output),
		})
	}

	return c.JSON(fiber.Map{"success": true, "message": "Image removed"})
}

// RunContainer creates and runs a new container
func RunContainer(c *fiber.Ctx) error {
	type RunRequest struct {
		Image   string            `json:"image"`
		Name    string            `json:"name"`
		Ports   map[string]string `json:"ports"`
		Env     map[string]string `json:"env"`
		Volumes map[string]string `json:"volumes"`
		Detach  bool              `json:"detach"`
	}

	var req RunRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Image == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Image is required"})
	}

	args := []string{"run"}

	if req.Detach {
		args = append(args, "-d")
	}

	if req.Name != "" {
		args = append(args, "--name", req.Name)
	}

	for host, container := range req.Ports {
		args = append(args, "-p", host+":"+container)
	}

	for key, value := range req.Env {
		args = append(args, "-e", key+"="+value)
	}

	for host, container := range req.Volumes {
		args = append(args, "-v", host+":"+container)
	}

	args = append(args, req.Image)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error":  "Failed to run container",
			"output": string(output),
		})
	}

	return c.JSON(fiber.Map{
		"success":      true,
		"message":      "Container created",
		"container_id": strings.TrimSpace(string(output)),
	})
}

// Unused import fix for runtime
var _ = runtime.GOOS
