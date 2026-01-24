package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config структура конфигурации
type Config struct {
	Port string
	// UploadDir  string
	// Update     string
	// BackupAPP  string
	// RestoreAPP string
	// BackupBD   string
	LimitMB   int64
	ApiPrefix string
}

// Response структура ответа
type Response struct {
	Success  int    `json:"success,omitempty"`
	Uploaded int    `json:"uploaded,omitempty"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Структура запроса
type Reguest struct {
	Timestamp string `json:"timestamp"` // пример: "2026-01-23T11:07:00+03:00"
}

// Получаем значение по умолчанию, если не заданны переменные окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

var cfg Config

const baseDir = "./files"
const port = ":8080"

func init() {

	// Проверяем корректность ввода значения limitMB
	limitMB, err := strconv.ParseInt(getEnv("UPT_LIMIT_DOWNLOAD_MB", "500"), 10, 64)
	if err != nil {
		log.Fatalf("❌ Не корректный формат UPT_LIMIT_DOWNLOAD_MB: %v", err)
		return
	}

	cfg = Config{
		LimitMB: limitMB,
		Port:    getEnv("UPT_PORT", ":8080"),
		// UploadDir:  getEnv("UPT_PATH_PREFIX", "./uploads"),
		// Update:     getEnv("UPT_SC_UPDATE", "lscpu"),
		ApiPrefix: getEnv("UPT_URL_API_PREFIX", ""),
		// BackupAPP:  getEnv("UPT_SC_BACKUP_APP", "who"),
		// RestoreAPP: getEnv("UPT_SC_RESTORE_APP", "vmstat"),
		// BackupBD:   getEnv("UPT_SC_BACKUP_BD", "lsblk"),
	}

	// if err := os.MkdirAll(cfg.UploadDir, 0750); err != nil {
	// 	log.Fatalf("❌ Ошибка создания директории: %v", err)
	// }
}

// registerRoute регистрирует обработчик и сразу выводит итоговый путь в лог
func registerRoute(pattern string, handler http.HandlerFunc) {
	http.HandleFunc(pattern, handler)
	log.Printf("🔖 EdnPoint зарегистрирован: %s", pattern)
}

func main() {

	// обработать ошибку существование директории
	_ = os.MkdirAll(baseDir, 0755)

	// http.HandleFunc("/api/download", handleDownload)

	registerRoute(cfg.ApiPrefix+"/api/catalina", catalinalog)
	registerRoute(cfg.ApiPrefix+"/api/universe", universelog)
	registerRoute(cfg.ApiPrefix+"/api/alltomcat", alltomcatlog)
	registerRoute(cfg.ApiPrefix+"/api/scaners", scanerslog)

	// Запуск HTTP сервера
	log.Printf("🚀 Сервер запущен на http://localhost:%s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Ошибка запуска сервера: %v", err)
	}

}

func parseRFC3339(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, errors.New("empty")
	}
	// Стандартный парсер для RFC3339 в Go использует константу time.RFC3339 [web:24].
	return time.Parse(time.RFC3339, s)
}

func validationReguest(w http.ResponseWriter, r *http.Request) (string, error) {

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return "", fmt.Errorf("method not allowed")
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// (Опционально) ограничить размер body, чтобы не приняли 2GB «лог» случайно/в атаке.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB

	dec := json.NewDecoder(r.Body)

	// 2) если клиент прислал неизвестное поле — вернуть 400.
	dec.DisallowUnknownFields()

	var in Reguest
	if err := dec.Decode(&in); err != nil {
		http.Error(w, "invalid_json Invalid JSON body: ", http.StatusBadRequest)
		return "", fmt.Errorf("invalid_json Invalid JSON body")
	}

	// 3) если после первого JSON-объекта в body ещё мусор/второй объект — считаем это ошибкой формата.
	if dec.More() {
		http.Error(w, "invalid_json Unexpected extra JSON content", http.StatusBadRequest)
		return "", fmt.Errorf("invalid_json Unexpected extra JSON content")
	}

	s, err := parseRFC3339(in.Timestamp)
	if err != nil {
		http.Error(w, "invalid_timestamp imestamp must be RFC3339, e.g. 2026-01-23T11:07:00+03:00", http.StatusBadRequest)
		return "", fmt.Errorf("invalid_timestamp imestamp must be RFC3339")
	}
	return s.Format(time.RFC3339), nil
}

func catalinalog(w http.ResponseWriter, r *http.Request) {

	ts, err := validationReguest(w, r)
	if err != nil {
		log.Printf("❌ Ошибка валидации JSON: %s", err)
		return
	}
	log.Printf("🚀 Timestamp: %v", ts)
}

func universelog(w http.ResponseWriter, r *http.Request) {

}

func alltomcatlog(w http.ResponseWriter, r *http.Request) {

}

func scanerslog(w http.ResponseWriter, r *http.Request) {

}

// func handleDownload(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != http.MethodPost {
// 		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
// 		return
// 	}

// 	// var req struct {
// 	// 	Files []struct {
// 	// 		Path string `json:"path"`
// 	// 		Type string `json:"type"`
// 	// 	} `json:"files"`
// 	// }

// 	var req Reguest

// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "bad request decode json", http.StatusBadRequest)
// 		return
// 	}

// 	w.Header().Set("Content-Type", "application/zip")
// 	w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

// 	zw := zip.NewWriter(w)
// 	defer zw.Close()

// 	base := filepath.Clean(baseDir)
// 	// debug
// 	// log.Printf("baseDir: %v", baseDir)
// 	// log.Printf("base: %v", base)

// 	for _, f := range req.Files {
// 		// проверка не пришёл ли путь выходящий за директорию с логами
// 		if !filepath.IsLocal(f.Path) {
// 			log.Printf("path escapes baseDir: path=%q ", f.Path)
// 			http.Error(w, "path escapes baseDir:", http.StatusInternalServerError)
// 			return
// 		}

// 		fullFilePath := filepath.Clean(filepath.Join(baseDir, f.Path))

// 		// debug
// 		// log.Printf("f.Path: %v", f.Path)
// 		// log.Printf("fullFilePath: %v", fullFilePath)

// 		if !strings.HasPrefix(fullFilePath, base) {
// 			continue
// 		}
// 		switch f.Type {
// 		case "file":
// 			if err := addFileToZip(zw, fullFilePath, f.Path); err != nil {
// 				log.Printf("addFileToZip failed: path=%q err=%v", f.Path, err)
// 				http.Error(w, "failed to add file to zip", http.StatusInternalServerError)
// 				return
// 			}
// 		case "directory":
// 			if err := addDirToZip(zw, fullFilePath, f.Path); err != nil {
// 				log.Printf("addFileToZip failed: path=%q err=%v", f.Path, err)
// 				http.Error(w, "failed to add Dir to zip", http.StatusInternalServerError)
// 				return
// 			}
// 		}
// 	}
// }

func addFileToZip(zw *zip.Writer, filePath, archivePath string) error {

	log.Printf("Zip: открытие файла %s", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	h, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	h.Name = strings.ReplaceAll(archivePath, "\\", "/")
	h.Method = zip.Deflate

	w, err := zw.CreateHeader(h)
	if err != nil {
		return err
	}

	n, err := io.Copy(w, file)
	if err != nil {
		return err
	}

	log.Printf("Zip: файл добавлен %s (%d байт записано)", h.Name, n)
	return nil
}

func addDirToZip(zw *zip.Writer, dirPath, archivePath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}
	for _, e := range entries {
		fullFilePath := filepath.Join(dirPath, e.Name())
		ap := strings.ReplaceAll(filepath.Join(archivePath, e.Name()), "\\", "/")
		if e.IsDir() {
			if err := addDirToZip(zw, fullFilePath, ap); err != nil {
				return err
			}
		} else {
			if err := addFileToZip(zw, fullFilePath, ap); err != nil {
				return err
			}
		}
	}
	return nil
}
