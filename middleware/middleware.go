package middleware

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func RegWebRouter(e *gin.Engine, content embed.FS) {

	distFS, _ := fs.Sub(content, "frontend/dist")
	// 静态文件处理
	e.Use(func(c *gin.Context) {

		path := c.Request.URL.Path

		// API不处理
		if strings.HasPrefix(path, "/api/") {
			c.Next()
			return
		}

		file, err := fs.Stat(distFS, path[1:])

		if err == nil && !file.IsDir() {

			http.FileServer(
				http.FS(distFS),
			).ServeHTTP(c.Writer, c.Request)

			c.Abort()
			return
		}

		c.Next()
	})
	e.NoRoute(func(context *gin.Context) {
		index, err := fs.ReadFile(distFS, "index.html")
		if err != nil {
			context.String(500, err.Error())
			return
		}
		context.Data(
			200,
			"text/html; charset=utf-8",
			index,
		)
	})
	log.Println("已开启前后端整合模式！")
}

func RegCORS(r *gin.Engine) {
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})
}
