package handlers

import (
	"encoding/json"
	"log"
	"music-download/model"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type DownloadTask struct {
	Music  *model.Music
	Cancel chan bool
}

var (
	musicList = make([]model.Music, 0)
	taskMap   = make(map[int]*DownloadTask)
	taskMutex sync.Mutex
	nextID    = 1
)

// 清理文件名，移除非法字符并确保有效
func sanitizeFileName(name string) string {
	// 替换非法字符
	illegalChars := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, c := range illegalChars {
		name = strings.ReplaceAll(name, c, "_")
	}
	// 移除前后的空格
	name = strings.TrimSpace(name)
	// 如果为空，提供默认名称
	if name == "" {
		name = "downloaded_file"
	}
	return name
}

// 从URL中提取文件扩展名
func getExtensionFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil {
		return ".flac"
	}
	path := u.Path
	ext := filepath.Ext(path)
	if ext == "" {
		return ".flac"
	}
	return ext
}

func GetMusicList(c *gin.Context) {
	taskMutex.Lock()
	defer taskMutex.Unlock()
	c.JSON(http.StatusOK, musicList)
}

func AddMusic(c *gin.Context) {
	var newMusic model.Music
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

func GetMusic(c *gin.Context) {
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

func DeleteMusic(c *gin.Context) {
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

func StartDownloadHandler(c *gin.Context) {
	var req struct {
		ID       int    `json:"id"`
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	taskMutex.Lock()
	var music *model.Music
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
		music.FileName = sanitizeFileName(req.FileName)
		if filepath.Ext(music.FileName) == "" {
			ext := getExtensionFromURL(music.URL)
			music.FileName += ext
		}
	} else {
		ext := getExtensionFromURL(music.URL)
		music.FileName = sanitizeFileName(music.Name) + ext
	}

	music.Status = "downloading"
	music.Progress = 0

	cancel := make(chan bool)
	taskMap[req.ID] = &DownloadTask{
		Music:  music,
		Cancel: cancel,
	}
	taskMutex.Unlock()

	go StartDownload(music, cancel)

	c.JSON(http.StatusOK, gin.H{"status": "started"})
}

func StartDownload(music *model.Music, cancel chan bool) {
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

func ProgressSSE(c *gin.Context) {
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

// 推送接口：接收音乐数据并直接开始下载
func PushAndDownload(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required"`
		URL      string `json:"url" binding:"required"`
		Time     string `json:"time"`
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log.Printf("Received push request: %v", input)
	if input.Time == "" {
		input.Time = time.Now().Format("2006/01/02 15:04:05")
	}

	// 如果没有提供文件名，生成一个
	if input.FileName == "" {
		ext := getExtensionFromURL(input.URL)
		input.FileName = sanitizeFileName(input.Name) + ext
	} else {
		input.FileName = sanitizeFileName(input.FileName)
		if filepath.Ext(input.FileName) == "" {
			ext := getExtensionFromURL(input.URL)
			input.FileName += ext
		}
	}

	newMusic := model.Music{
		Name:       input.Name,
		URL:        input.URL,
		Time:       input.Time,
		FileName:   input.FileName,
		Downloaded: false,
		Progress:   0,
		Status:     "pending",
	}

	taskMutex.Lock()
	newMusic.ID = nextID
	nextID++
	musicList = append(musicList, newMusic)
	taskMutex.Unlock()

	// 启动下载
	cancel := make(chan bool)
	taskMutex.Lock()
	taskMap[newMusic.ID] = &DownloadTask{
		Music:  &musicList[len(musicList)-1],
		Cancel: cancel,
	}
	taskMutex.Unlock()

	go StartDownload(&musicList[len(musicList)-1], cancel)

	c.JSON(http.StatusOK, gin.H{
		"message": "音乐已添加并开始下载",
		"music":   newMusic,
	})
}

// 批量推送并下载接口
func BatchPushAndDownload(c *gin.Context) {
	var songs []struct {
		Name     string `json:"name" binding:"required"`
		URL      string `json:"url" binding:"required"`
		Time     string `json:"time"`
		FileName string `json:"fileName"`
	}

	if err := c.ShouldBindJSON(&songs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 第一步：先添加所有歌曲到 musicList（一次性扩容，避免指针失效）
	taskMutex.Lock()
	startIndex := len(musicList)
	for _, song := range songs {
		if song.Time == "" {
			song.Time = time.Now().Format("2006/01/02 15:04:05")
		}

		// 处理文件名
		if song.FileName == "" {
			ext := getExtensionFromURL(song.URL)
			song.FileName = sanitizeFileName(song.Name) + ext
		} else {
			song.FileName = sanitizeFileName(song.FileName)
			if filepath.Ext(song.FileName) == "" {
				ext := getExtensionFromURL(song.URL)
				song.FileName += ext
			}
		}

		newMusic := model.Music{
			ID:         nextID,
			Name:       song.Name,
			URL:        song.URL,
			Time:       song.Time,
			FileName:   song.FileName,
			Downloaded: false,
			Progress:   0,
			Status:     "pending",
		}
		nextID++
		musicList = append(musicList, newMusic)
	}
	taskMutex.Unlock()

	// 第二步：再启动所有下载（此时切片已经稳定，指针有效）
	addedMusic := make([]model.Music, 0, len(songs))
	for i := 0; i < len(songs); i++ {
		index := startIndex + i
		taskMutex.Lock()
		music := &musicList[index]
		addedMusic = append(addedMusic, *music)
		cancel := make(chan bool)
		taskMap[music.ID] = &DownloadTask{
			Music:  music,
			Cancel: cancel,
		}
		taskMutex.Unlock()

		go StartDownload(music, cancel)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "批量音乐已添加并开始下载",
		"count":     len(addedMusic),
		"musicList": addedMusic,
	})
}
