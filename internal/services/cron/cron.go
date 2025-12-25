package cron

import (
	"log"
	"os/exec"
	"runtime"
	"sync"
	"time"

	"vps-panel/internal/database"
	"vps-panel/internal/models"

	"github.com/robfig/cron/v3"
)

var (
	cronScheduler *cron.Cron
	jobMap        map[uint]cron.EntryID
	mutex         sync.Mutex
)

// Init initializes the cron scheduler
func Init() {
	cronScheduler = cron.New()
	jobMap = make(map[uint]cron.EntryID)

	// Migrate database
	database.DB.AutoMigrate(&models.CronJob{})

	// Load existing jobs
	var jobs []models.CronJob
	database.DB.Where("enabled = ?", true).Find(&jobs)

	for _, job := range jobs {
		addJobToScheduler(job)
	}

	cronScheduler.Start()
	log.Println("⏰ Cron scheduler started")
}

// AddJob adds a new cron job
func AddJob(name, schedule, command string) (*models.CronJob, error) {
	job := &models.CronJob{
		Name:     name,
		Schedule: schedule,
		Command:  command,
		Enabled:  true,
	}

	if err := database.DB.Create(job).Error; err != nil {
		return nil, err
	}

	addJobToScheduler(*job)
	return job, nil
}

// RemoveJob removes a cron job
func RemoveJob(id uint) error {
	mutex.Lock()
	entryID, exists := jobMap[id]
	if exists {
		cronScheduler.Remove(entryID)
		delete(jobMap, id)
	}
	mutex.Unlock()

	return database.DB.Delete(&models.CronJob{}, id).Error
}

// ToggleJob enables or disables a job
func ToggleJob(id uint, enabled bool) error {
	var job models.CronJob
	if err := database.DB.First(&job, id).Error; err != nil {
		return err
	}

	job.Enabled = enabled
	if err := database.DB.Save(&job).Error; err != nil {
		return err
	}

	mutex.Lock()
	entryID, exists := jobMap[id]
	if exists {
		cronScheduler.Remove(entryID)
		delete(jobMap, id)
	}
	mutex.Unlock()

	if enabled {
		addJobToScheduler(job)
	}

	return nil
}

// GetJobs returns all jobs
func GetJobs() ([]models.CronJob, error) {
	var jobs []models.CronJob
	err := database.DB.Order("created_at desc").Find(&jobs).Error
	return jobs, err
}

func addJobToScheduler(job models.CronJob) {
	entryID, err := cronScheduler.AddFunc(job.Schedule, func() {
		runJob(job.ID, job.Command)
	})

	if err != nil {
		log.Printf("Failed to schedule job %s: %v", job.Name, err)
		return
	}

	mutex.Lock()
	jobMap[job.ID] = entryID
	mutex.Unlock()
}

func runJob(id uint, command string) {
	log.Printf("Running cron job %d: %s", id, command)

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	output, err := cmd.CombinedOutput()
	status := "success"
	result := string(output)

	if err != nil {
		status = "error"
		result += "\nError: " + err.Error()
	}

	// Update DB (in a separate goroutine to not block)
	go func() {
		now := time.Now()
		database.DB.Model(&models.CronJob{}).Where("id = ?", id).Updates(map[string]interface{}{
			"last_run":    now,
			"last_status": status,
			"last_result": result,
		})
	}()
}
