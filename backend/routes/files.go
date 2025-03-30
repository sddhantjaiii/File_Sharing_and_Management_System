package routes

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"filesharing/models"
	"filesharing/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func generateShareToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func UploadFile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user from context (set by auth middleware)
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
			return
		}

		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := time.Now().Format("20060102150405") + ext
		filepath := filepath.Join("uploads", filename)

		// Save file
		if err := c.SaveUploadedFile(file, filepath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
			return
		}

		// Generate share token
		shareToken, err := generateShareToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate share token"})
			return
		}

		// Create file record in database
		fileRecord := models.File{
			UserID:       userID.(uint),
			Filename:     filename,
			OriginalName: file.Filename,
			Size:         file.Size,
			MimeType:     file.Header.Get("Content-Type"),
			ShareToken:   shareToken,
		}

		if err := db.Create(&fileRecord).Error; err != nil {
			// Clean up uploaded file if database insert fails
			os.Remove(filepath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
			return
		}

		// Invalidate user's file cache
		utils.ClearUserCache(userID.(uint))

		c.JSON(http.StatusCreated, gin.H{
			"message": "File uploaded successfully",
			"file":    fileRecord,
		})
	}
}

func ListFiles(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Try to get from cache first
		var files []models.File
		cacheKey := "user:files:" + string(userID.(uint))
		if err := utils.GetCache(cacheKey, &files); err == nil {
			c.JSON(http.StatusOK, gin.H{"files": files})
			return
		}

		// If not in cache, get from database
		if err := db.Where("user_id = ?", userID).Find(&files).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
			return
		}

		// Cache the results
		utils.SetCache(cacheKey, files, 5*time.Minute)

		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func ShareFile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("file_id")
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Validate file ID
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
			return
		}

		// Convert file ID to uint
		fileIDUint, err := strconv.ParseUint(fileID, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID format"})
			return
		}

		var file models.File
		if err := db.Where("id = ? AND user_id = ?", uint(fileIDUint), userID).First(&file).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
			return
		}

		// Generate new share token if not exists
		if file.ShareToken == "" {
			shareToken, err := generateShareToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate share token"})
				return
			}
			file.ShareToken = shareToken
			if err := db.Save(&file).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save share token"})
				return
			}
		}

		// Get scheme and host from request
		scheme := "http"
		if c.Request.TLS != nil {
			scheme = "https"
		}
		host := c.Request.Host

		// Generate share URL - using the correct path format
		shareURL := fmt.Sprintf("%s://%s/api/files/shared/%s", scheme, host, file.ShareToken)

		c.JSON(http.StatusOK, gin.H{
			"share_url": shareURL,
			"file": file,
		})
	}
}

func SearchFiles(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		query := c.Query("query")
		if query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
			return
		}

		var files []models.File
		if err := db.Where("user_id = ? AND original_name LIKE ?", userID, "%"+query+"%").Find(&files).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search files"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"files": files})
	}
}

func DeleteFile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		fileID := c.Param("file_id")
		if fileID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "File ID is required"})
			return
		}

		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		var file models.File
		if err := db.Where("id = ? AND user_id = ?", fileID, userID).First(&file).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
			}
			return
		}

		// Delete physical file
		filepath := filepath.Join("uploads", file.Filename)
		if err := os.Remove(filepath); err != nil {
			if !os.IsNotExist(err) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
				return
			}
			// If file doesn't exist, continue with database deletion
		}

		// Delete database record
		if err := db.Delete(&file).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file record"})
			return
		}

		// Invalidate user's file cache
		utils.ClearUserCache(userID.(uint))

		c.JSON(http.StatusOK, gin.H{
			"message": "File deleted successfully",
			"id": fileID,
		})
	}
}

func GetSharedFile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		shareToken := c.Param("token")
		if shareToken == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Share token is required"})
			return
		}

		var file models.File
		if err := db.Where("share_token = ?", shareToken).First(&file).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				c.JSON(http.StatusNotFound, gin.H{"error": "File not found or link has expired"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file"})
			return
		}

		filePath := filepath.Join("uploads", file.Filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Set appropriate headers for file download
		c.Header("Content-Description", "File Transfer")
		c.Header("Content-Transfer-Encoding", "binary")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.OriginalName))
		c.Header("Content-Type", file.MimeType)
		c.File(filePath)
	}
} 