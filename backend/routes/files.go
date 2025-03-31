package routes

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"filesharing/models"
	"filesharing/utils"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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

		// Open the uploaded file
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
			return
		}
		defer src.Close()

		// Create channels for goroutine communication
		uploadChan := make(chan string, 1)
		dbChan := make(chan error, 1)
		errChan := make(chan error, 1)

		// Initialize S3 client
		s3Client, err := utils.NewS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize S3 client"})
			return
		}

		// Goroutine for S3 upload
		go func() {
			url, err := s3Client.UploadFile(c.Request.Context(), src, filename, file.Header.Get("Content-Type"))
			if err != nil {
				errChan <- fmt.Errorf("S3 upload failed: %v", err)
				return
			}
			uploadChan <- url
		}()

		// Goroutine for database operation
		go func() {
			shareToken, err := generateShareToken()
			if err != nil {
				errChan <- fmt.Errorf("failed to generate share token: %v", err)
				return
			}

			fileRecord := models.File{
				UserID:       userID.(uint),
				Filename:     filename,
				OriginalName: file.Filename,
				Size:         file.Size,
				MimeType:     file.Header.Get("Content-Type"),
				ShareToken:   shareToken,
				CreatedAt:    time.Now(),
			}

			if err := db.Create(&fileRecord).Error; err != nil {
				errChan <- fmt.Errorf("database operation failed: %v", err)
				return
			}
			dbChan <- nil
		}()

		// Wait for both operations to complete
		select {
		case err := <-errChan:
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		case <-time.After(30 * time.Second):
			c.JSON(http.StatusRequestTimeout, gin.H{"error": "Upload timeout"})
			return
		case url := <-uploadChan:
			if err := <-dbChan; err != nil {
				// If database operation fails, try to delete the uploaded file
				go s3Client.DeleteFile(c.Request.Context(), filename)
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			// Fetch the created file record
			var fileRecord models.File
			if err := db.Where("filename = ?", filename).First(&fileRecord).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch file record"})
				return
			}

			// Format file for response
			formattedFile := gin.H{
				"ID":            fileRecord.ID,
				"CreatedAt":     fileRecord.CreatedAt,
				"UpdatedAt":     fileRecord.UpdatedAt,
				"DeletedAt":     fileRecord.DeletedAt,
				"user_id":       fileRecord.UserID,
				"filename":      fileRecord.Filename,
				"original_name": fileRecord.OriginalName,
				"size":          fileRecord.Size,
				"mime_type":     fileRecord.MimeType,
				"share_token":   fileRecord.ShareToken,
				"share_url":     url,
			}

			// Invalidate the cache for this user's files
			cacheKey := "user:files:" + string(userID.(uint))
			utils.DeleteCache(cacheKey)

			c.JSON(http.StatusOK, gin.H{
				"message": "File uploaded successfully",
				"file":    formattedFile,
			})
		}
	}
}

func ListFiles(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		// Get files from database first
		var files []models.File
		if err := db.Where("user_id = ?", userID).Find(&files).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
			return
		}

		// Initialize S3 client
		s3Client, err := utils.NewS3Client()
		if err != nil {
			log.Printf("Error initializing S3 client: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize S3 client"})
			return
		}

		// Check if each file exists in S3 and generate presigned URLs
		var validFiles []models.File
		for _, file := range files {
			key := fmt.Sprintf("uploads/%s/%s", file.CreatedAt.Format("2006/01/02"), file.Filename)
			if err := s3Client.HeadObject(c.Request.Context(), key); err == nil {
				// Generate presigned URL for the file
				presignClient := s3.NewPresignClient(s3Client.Client)
				presignResult, err := presignClient.PresignGetObject(c.Request.Context(), &s3.GetObjectInput{
					Bucket: aws.String(s3Client.Bucket),
					Key:    aws.String(key),
				}, func(opts *s3.PresignOptions) {
					opts.Expires = time.Hour * 24 * 7 // 7 days
				})
				if err == nil {
					file.ShareToken = presignResult.URL // Use ShareToken instead of ShareURL
					validFiles = append(validFiles, file)
				}
			} else {
				// File doesn't exist in S3, delete it from database
				db.Delete(&file)
			}
		}

		// Cache the valid files
		cacheKey := fmt.Sprintf("user:files:%d", userID)
		if validFilesJSON, err := json.Marshal(validFiles); err == nil {
			utils.SetCache(cacheKey, string(validFilesJSON), time.Hour)
		}

		c.JSON(http.StatusOK, gin.H{"files": validFiles})
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

		// Initialize S3 client
		s3Client, err := utils.NewS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize S3 client"})
			return
		}

		// Generate presigned URL
		url, err := s3Client.UploadFile(c.Request.Context(), nil, file.Filename, file.MimeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
			return
		}

		// Format file for response
		formattedFile := gin.H{
			"ID":            file.ID,
			"CreatedAt":     file.CreatedAt,
			"UpdatedAt":     file.UpdatedAt,
			"DeletedAt":     file.DeletedAt,
			"user_id":       file.UserID,
			"filename":      file.Filename,
			"original_name": file.OriginalName,
			"size":          file.Size,
			"mime_type":     file.MimeType,
			"share_url":     url,
		}

		c.JSON(http.StatusOK, gin.H{
			"share_url": url,
			"file":      formattedFile,
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
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			return
		}

		fileID := c.Param("file_id")
		// Convert file ID to uint
		fileIDUint, err := strconv.ParseUint(fileID, 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file ID format"})
			return
		}

		var file models.File
		if err := db.Where("id = ? AND user_id = ?", uint(fileIDUint), userID).First(&file).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Initialize S3 client
		s3Client, err := utils.NewS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize S3 client"})
			return
		}

		// Delete from S3
		if err := s3Client.DeleteFile(c.Request.Context(), file.Filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from S3"})
			return
		}

		// Delete from database
		if err := db.Delete(&file).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file from database"})
			return
		}

		// Invalidate cache
		cacheKey := fmt.Sprintf("files:%d", userID)
		utils.DeleteCache(cacheKey)

		c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
	}
}

func GetSharedFile(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Param("token")
		var file models.File
		if err := db.Where("share_token = ?", token).First(&file).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
			return
		}

		// Initialize S3 client
		s3Client, err := utils.NewS3Client()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initialize S3 client"})
			return
		}

		// Generate presigned URL
		url, err := s3Client.UploadFile(c.Request.Context(), nil, file.Filename, file.MimeType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate download URL"})
			return
		}

		// Format file for response
		formattedFile := gin.H{
			"ID":            file.ID,
			"CreatedAt":     file.CreatedAt,
			"UpdatedAt":     file.UpdatedAt,
			"DeletedAt":     file.DeletedAt,
			"user_id":       file.UserID,
			"filename":      file.Filename,
			"original_name": file.OriginalName,
			"size":          file.Size,
			"mime_type":     file.MimeType,
			"share_url":     url,
		}

		c.JSON(http.StatusOK, gin.H{
			"file": formattedFile,
			"url":  url,
		})
	}
}
