package main

import (
	"embed"
	"log"
	"music-download/middleware"
	"music-download/router"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist
var embedFS embed.FS

func main() {
	r := gin.Default()

	// CORS middleware
	middleware.RegCORS(r)
	middleware.RegWebRouter(r, embedFS)
	// API routes
	router.RegisterRouter(r)

	log.Println("Server starting on :8080...")
	r.Run(":8080")
}
