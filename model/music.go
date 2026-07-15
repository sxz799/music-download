package model

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
