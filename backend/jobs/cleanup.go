package jobs

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"filesharing/models"
	"filesharing/utils"

	"gorm.io/gorm"
)

func StartCleanupJob(db *gorm.DB) {
	go func() {
		for {
			// Find expired files
			var expiredFiles []models.File
			if err := db.Where("expires_at < ?", time.Now()).Find(&expiredFiles).Error; err != nil {
				log.Printf("Error finding expired files: %v", err)
				time.Sleep(time.Hour)
				continue
			}

			// Delete expired files
			for _, file := range expiredFiles {
				// Delete physical file
				filepath := filepath.Join("uploads", file.Filename)
				if err := os.Remove(filepath); err != nil {
					log.Printf("Error deleting expired file %s: %v", file.Filename, err)
					continue
				}

				// Delete database record
				if err := db.Delete(&file).Error; err != nil {
					log.Printf("Error deleting expired file record %d: %v", file.ID, err)
					continue
				}

				// Invalidate user's file cache
				utils.ClearUserCache(file.UserID)

				log.Printf("Deleted expired file: %s", file.OriginalName)
			}

			// Sleep for an hour before next cleanup
			time.Sleep(time.Hour)
		}
	}()
} 