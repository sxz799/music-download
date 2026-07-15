package router

import (
	handlers "music-download/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRouter(r *gin.Engine) {
	// API routes
	r.GET("/api/music", handlers.GetMusicList)
	r.POST("/api/music", handlers.AddMusic)
	r.GET("/api/music/:id", handlers.GetMusic)
	r.DELETE("/api/music/:id", handlers.DeleteMusic)
	r.POST("/api/download", handlers.StartDownloadHandler)
	r.GET("/api/progress", handlers.ProgressSSE)

	// 推送接口
	r.POST("/api/push", handlers.PushAndDownload)
	r.POST("/api/push/batch", handlers.BatchPushAndDownload)
}
