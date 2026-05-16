package schedule

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

const yadiskAPI = "https://cloud-api.yandex.net/v1/disk/public/resources"

type FileInfo struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Modified time.Time `json:"modified"`
	Path     string    `json:"path"`
	Type     string    `json:"type"`
}

type Downloader struct {
	scheduleDir      string
	substitutionsDir string
	scheduleKey      string
	substitutionsKey string
	intervalHours    int
}

func New(scheduleDir, substitutionsDir, scheduleKey, substitutionsKey string, intervalHours int) *Downloader {
	return &Downloader{
		scheduleDir:      scheduleDir,
		substitutionsDir: substitutionsDir,
		scheduleKey:      scheduleKey,
		substitutionsKey: substitutionsKey,
		intervalHours:    intervalHours,
	}
}

func (d *Downloader) Start() {
	go func() {
		d.sync()
		ticker := time.NewTicker(time.Duration(d.intervalHours) * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			d.sync()
		}
	}()
}

func (d *Downloader) SyncNow() { d.sync() }

func (d *Downloader) sync() {
	log.Println("[schedule] синхронизация расписания...")
	if err := d.syncDir(d.scheduleKey, d.scheduleDir); err != nil {
		log.Printf("[schedule] ошибка расписания: %v", err)
	}
	if err := d.syncDir(d.substitutionsKey, d.substitutionsDir); err != nil {
		log.Printf("[schedule] ошибка замен: %v", err)
	}
	log.Println("[schedule] синхронизация завершена")
}

func (d *Downloader) syncDir(publicKey, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	items, err := listPublicDir(publicKey)
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	for _, item := range items {
		destPath := filepath.Join(destDir, item.Name)
		if _, err := os.Stat(destPath); err == nil {
			continue
		}
		if err := downloadFile(item.DownloadURL, destPath); err != nil {
			log.Printf("[schedule] не удалось скачать %s: %v", item.Name, err)
			continue
		}
		log.Printf("[schedule] скачан: %s", item.Name)
	}
	return nil
}

type yadiskItem struct {
	Name        string `json:"name"`
	DownloadURL string `json:"file"`
	MediaType   string `json:"media_type"`
	MimeType    string `json:"mime_type"`
}

type yadiskResp struct {
	Embedded struct {
		Items []yadiskItem `json:"items"`
	} `json:"_embedded"`
}

func listPublicDir(publicKey string) ([]yadiskItem, error) {
	params := url.Values{}
	params.Set("public_key", publicKey)
	params.Set("limit", "100")
	params.Set("fields", "_embedded.items.name,_embedded.items.file,_embedded.items.media_type,_embedded.items.mime_type")

	resp, err := http.Get(yadiskAPI + "?" + params.Encode())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API %d: %s", resp.StatusCode, string(body))
	}

	var data yadiskResp
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Embedded.Items, nil
}

func downloadFile(fileURL, destPath string) error {
	resp, err := http.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func ListFiles(dir string) ([]FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []FileInfo{}, nil
		}
		return nil, err
	}

	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		ext := filepath.Ext(e.Name())
		ftype := extToType(ext)
		files = append(files, FileInfo{
			Name:     e.Name(),
			Size:     info.Size(),
			Modified: info.ModTime(),
			Path:     "/files/" + filepath.Base(dir) + "/" + e.Name(),
			Type:     ftype,
		})
	}
	return files, nil
}

func extToType(ext string) string {
	switch ext {
	case ".pdf":
		return "pdf"
	case ".doc", ".docx":
		return "word"
	case ".jpg", ".jpeg", ".png", ".webp":
		return "image"
	case ".xls", ".xlsx":
		return "excel"
	default:
		return "file"
	}
}
