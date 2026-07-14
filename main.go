
package main

import (
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

//go:embed frontend/dist
var embedFS embed.FS

type Music struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	Time       string `json:"time"`
	Downloaded bool   `json:"downloaded"`
	FileName   string `json:"fileName"`
	Progress   int    `json:"progress"`
	Status     string `json:"status"`
}

type DownloadTask struct {
	Music  *Music
	Cancel chan bool
}

var (
	musicList  []Music
	taskMap    = make(map[int]*DownloadTask)
	taskMutex  sync.Mutex
	nextID     = 1
)

func main() {
	r := gin.Default()

	// CORS middleware
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

	// API routes
	r.GET("/api/music", getMusicList)
	r.POST("/api/music", addMusic)
	r.GET("/api/music/:id", getMusic)
	r.DELETE("/api/music/:id", deleteMusic)
	r.POST("/api/download", startDownloadHandler)
	r.GET("/api/progress", progressSSE)

	// Try to use embed first, if failed use local file system
	var staticHandler gin.HandlerFunc
	staticFS, err := fs.Sub(embedFS, "frontend/dist")
	if err != nil {
		log.Printf("Embed frontend not found, using local filesystem: %v", err)
		// Check if local dist exists
		if _, err := os.Stat("frontend/dist"); os.IsNotExist(err) {
			log.Printf("Local frontend/dist not found either, serving API only")
		} else {
			staticHandler = gin.WrapH(http.FileServer(http.Dir("frontend/dist")))
		}
	} else {
		log.Println("Using embedded frontend")
		staticHandler = gin.WrapH(http.FileServer(http.FS(staticFS)))
	}

	// Handle frontend routes with NoRoute
	if staticHandler != nil {
		r.NoRoute(func(c *gin.Context) {
			// If it's not an API route, serve static file
			staticHandler(c)
		})
	}

	log.Println("Server starting on :8080...")
	r.Run(":8080")
}

func getMusicList(c *gin.Context) {
	taskMutex.Lock()
	defer taskMutex.Unlock()
	c.JSON(http.StatusOK, musicList)
}

func addMusic(c *gin.Context) {
	var newMusic Music
	if err := c.ShouldBindJSON(&newMusic); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskMutex.Lock()
	newMusic.ID = nextID
	newMusic.Progress = 0
	newMusic.Status = "pending"
	nextID++
	musicList = append(musicList, newMusic)
	taskMutex.Unlock()

	c.JSON(http.StatusCreated, newMusic)
}

func getMusic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	taskMutex.Lock()
	defer taskMutex.Unlock()

	for _, m := range musicList {
		if m.ID == id {
			c.JSON(http.StatusOK, m)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Music not found"})
}

func deleteMusic(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	taskMutex.Lock()
	defer taskMutex.Unlock()

	if task, exists := taskMap[id]; exists {
		close(task.Cancel)
		delete(taskMap, id)
	}

	for i, m := range musicList {
		if m.ID == id {
			musicList = append(musicList[:i], musicList[i+1:]...)
			c.Status(http.StatusNoContent)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Music not found"})
}

func startDownloadHandler(c *gin.Context) {
	var req struct {
		ID       int    `json:"id"`
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskMutex.Lock()
	var music *Music
	for i := range musicList {
		if musicList[i].ID == req.ID {
			music = &musicList[i]
			break
		}
	}

	if music == nil {
		taskMutex.Unlock()
		c.JSON(http.StatusNotFound, gin.H{"error": "Music not found"})
		return
	}

	if _, exists := taskMap[req.ID]; exists {
		taskMutex.Unlock()
		c.JSON(http.StatusBadRequest, gin.H{"error": "Already downloading"})
		return
	}

	if req.FileName != "" {
		music.FileName = req.FileName
	} else {
		music.FileName = music.Name + filepath.Ext(music.URL)
	}

	music.Status = "downloading"
	music.Progress = 0

	cancel := make(chan bool)
	taskMap[req.ID] = &DownloadTask{
		Music:  music,
		Cancel: cancel,
	}
	taskMutex.Unlock()

	go startDownload(music, cancel)

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func startDownload(music *Music, cancel chan bool) {
	defer func() {
		taskMutex.Lock()
		delete(taskMap, music.ID)
		taskMutex.Unlock()
	}()

	resp, err := http.Get(music.URL)
	if err != nil {
		taskMutex.Lock()
		music.Status = "error"
		taskMutex.Unlock()
		log.Printf("Download error for %s: %v", music.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		taskMutex.Lock()
		music.Status = "error"
		taskMutex.Unlock()
		log.Printf("Download failed for %s: status %d", music.Name, resp.StatusCode)
		return
	}

	contentLength := resp.ContentLength

	downloadDir := "./downloads"
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		taskMutex.Lock()
		music.Status = "error"
		taskMutex.Unlock()
		log.Printf("Create download dir error: %v", err)
		return
	}

	filePath := filepath.Join(downloadDir, music.FileName)
	out, err := os.Create(filePath)
	if err != nil {
		taskMutex.Lock()
		music.Status = "error"
		taskMutex.Unlock()
		log.Printf("Create file error: %v", err)
		return
	}
	defer out.Close()

	buf := make([]byte, 32*1024)
	var downloaded int64

	for {
		select {
		case <-cancel:
			taskMutex.Lock()
			music.Status = "cancelled"
			taskMutex.Unlock()
			out.Close()
			os.Remove(filePath)
			log.Printf("Download cancelled for %s", music.Name)
			return
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				taskMutex.Lock()
				music.Status = "error"
				taskMutex.Unlock()
				log.Printf("Write file error: %v", writeErr)
				return
			}
			downloaded += int64(n)

			if contentLength > 0 {
				taskMutex.Lock()
				music.Progress = int(float64(downloaded) / float64(contentLength) * 100)
				taskMutex.Unlock()
			}
		}

		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			taskMutex.Lock()
			music.Status = "error"
			taskMutex.Unlock()
			log.Printf("Read response error: %v", err)
			return
		}
	}

	taskMutex.Lock()
	music.Downloaded = true
	music.Status = "completed"
	music.Progress = 100
	taskMutex.Unlock()

	log.Printf("Download completed: %s", music.FileName)
}

func progressSSE(c *gin.Context) {
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			taskMutex.Lock()
			data, _ := json.Marshal(musicList)
			taskMutex.Unlock()

			c.SSEvent("message", string(data))
			c.Writer.Flush()
		}
	}
}

func init() {
	data, err := os.ReadFile("music.json")
	if err == nil {
		var loaded []Music
		if json.Unmarshal(data, &loaded) == nil {
			musicList = loaded
			for i := range musicList {
				if musicList[i].ID >= nextID {
					nextID = musicList[i].ID + 1
				}
			}
		}
	}
}
