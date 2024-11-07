package main

import (
	"go-restaurant-managament/database"
	middleware "go-restaurant-managament/middleware"
	routes "go-restaurant-managament/routes"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
)

var foodCollection *mongo.Collection = database.OpenCollection(database.Client, "food")

func main() {
	dir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Current working directory: %s", dir)

	_, err = os.Stat(".env")
	if os.IsNotExist(err) {
		log.Println(".env file does not exist in the current working directory")
	} else if err != nil {
		log.Println("Error checking .env file:", err)
	} else {
		log.Println(".env file found")
	}

	log.Println("Looking for .env file...")
	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	router := gin.New()
	router.Use(gin.Logger())

	// Enable CORS
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:9000"}, // Change this to your frontend URL if needed
		// AllowOrigins:     []string{"http://your-frontend-url.com"}, // Allow requests from your frontend URL
		AllowMethods:     []string{"POST", "GET", "PATCH", "DELETE", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Serve Vue.js static files
	router.Static("/frontend", filepath.Join(".", "frontend", "dist"))
	// router.Static("/frontend", "/home/user/go-development/restaurant/restaurant-management-frontend/dist")
	// router.Static("/config", "./config")
	// Fallback route to serve index.html for Vue.js frontend routes
	router.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/frontend") {
			// Serve index.html for frontend routes
			c.File(filepath.Join(".", "frontend", "dist", "index.html"))
			// c.File(filepath.Join("/home/user/go-development/restaurant/restaurant-management-frontend/dist", "index.html"))
		} else {
			c.JSON(http.StatusNotFound, gin.H{"message": "Page not found"})
		}
	})

	// API routes
	routes.UserRoutes(router)
	router.Use(middleware.Authentication())
	routes.FoodRoutes(router)
	routes.MenuRoutes(router)
	routes.TableRoutes(router)
	routes.OrderRoutes(router)
	routes.OrderItemRoutes(router)
	routes.InvoiceRoutes(router)

	// Run the server
	router.Run(":" + port)
}
