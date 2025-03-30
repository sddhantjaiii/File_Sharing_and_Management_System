package main

import (
	"fmt"
	"log"
	"os"

	"filesharing/jobs"
	"filesharing/middleware"
	"filesharing/models"
	"filesharing/routes"
	"filesharing/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"github.com/gin-contrib/cors"
)

func initDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&models.User{}, &models.File{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found")
	}

	// Initialize database
	db, err := initDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Initialize Redis
	if err := utils.InitRedis(); err != nil {
		log.Fatal("Failed to connect to Redis:", err)
	}

	// Create uploads directory if it doesn't exist
	if err := os.MkdirAll("./uploads", 0755); err != nil {
		log.Fatal("Failed to create uploads directory:", err)
	}

	// Initialize Gin router
	r := gin.Default()

	// CORS configuration
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://frontend:3000"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	config.AllowCredentials = true

	// CORS middleware
	r.Use(cors.New(config))

	// Serve static files from uploads directory
	r.Static("/uploads", "./uploads")

	// Initialize routes
	initializeRoutes(r, db)

	// Start cleanup job
	jobs.StartCleanupJob(db)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s...", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func initializeRoutes(r *gin.Engine, db *gorm.DB) {
	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	// Auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", routes.Register(db))
		auth.POST("/login", routes.Login(db))
	}

	// API routes
	api := r.Group("/api")
	{
		// Public route for accessing shared files
		api.GET("/files/shared/:token", routes.GetSharedFile(db))

		// Protected file routes
		files := api.Group("/files")
		files.Use(middleware.AuthMiddleware())
		{
			files.POST("/upload", routes.UploadFile(db))
			files.GET("", routes.ListFiles(db))
			files.GET("/search", routes.SearchFiles(db))
			files.GET("/share/:file_id", routes.ShareFile(db))
			files.DELETE("/:file_id", routes.DeleteFile(db))
		}
	}
} 