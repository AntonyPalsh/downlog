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
	Port            string
	LimitMB         int64
	ApiPrefix       string
	PathLogCatalina string
	PathLogUnivers  string
	PathLogScaners  string
	PathLogTomcat   string
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

// const baseDir = "./files"
// const port = ":8080"

func init() {

	// Проверяем корректность ввода значения limitMB
	limitMB, err := strconv.ParseInt(getEnv("UPT_LIMIT_DOWNLOAD_MB", "500"), 10, 64)
	if err != nil {
		log.Fatalf("❌ Не корректный формат UPT_LIMIT_DOWNLOAD_MB: %v", err)
		return
	}

	cfg = Config{
		LimitMB:         limitMB,
		Port:            getEnv("DL_PORT", ":8080"),
		ApiPrefix:       getEnv("DL_URL_API_PREFIX", ""),
		PathLogCatalina: getEnv("DL_CATALINA_LOG", "/app/edm/tomcat-9/logs/catalina"),
		PathLogUnivers:  getEnv("DL_UNIVERS_LOG", "closed/universe_backend"),
		PathLogScaners:  getEnv("DL_SCAN_LOG", "/app/edm/scan/logs"),
		PathLogTomcat:   getEnv("DL_TOMCAT", "/app/edm/tomcat-9/logs"),
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
	// _ = os.MkdirAll(baseDir, 0755)

	// http.HandleFunc("/api/download", handleDownload)

	registerRoute(cfg.ApiPrefix+"/api/catalina", catalinalog)
	registerRoute(cfg.ApiPrefix+"/api/universe", universelog)
	registerRoute(cfg.ApiPrefix+"/api/alltomcat", alltomcatlog)
	registerRoute(cfg.ApiPrefix+"/api/scaners", scanerslog)

	// Запуск HTTP сервера
	log.Printf("🚀 Сервер запущен на http://localhost:%s", cfg.Port)

	if err := http.ListenAndServe(cfg.Port, nil); err != nil {
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
	// преобразуем TimeStamp к виду в котором сохраняет файловая система и отправляем на выввод функции
	return s.Format("2006-01-02"), nil
}

func catalinalog(w http.ResponseWriter, r *http.Request) {

	ts, err := validationReguest(w, r)
	if err != nil {
		log.Printf("❌ Ошибка валидации JSON: %s", err)
		return
	}
	log.Printf("🪤 Timestamp: %v", ts)

	// Пример поиска файлов, измененных 26.01.2026, содержащих "log" в названии
	files, err := findFiles(ts, "/var/log", "auth.log")
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}

	fmt.Println("Найденные файлы:")
	for _, f := range files {
		fmt.Println(f)
	}

	handleDownload(w, files, "file")
}

func universelog(w http.ResponseWriter, r *http.Request) {

}

func alltomcatlog(w http.ResponseWriter, r *http.Request) {

}

func scanerslog(w http.ResponseWriter, r *http.Request) {

}

func handleDownload(w http.ResponseWriter, files []string, typef string) {

	fmt.Println("вызов функции handleDownload")

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment")

	zw := zip.NewWriter(w)

	for _, f := range files {
		switch typef {
		case "file":
			if err := addFileToZip(zw, f); err != nil {
				log.Printf("addFileToZip failed: path=%q err=%v", f, err)
				http.Error(w, "failed to add file to zip", http.StatusInternalServerError)
				return
			}
		case "dir":
			if err := addDirToZip(zw, f, f); err != nil {
				log.Printf("addFileToZip failed: path=%q err=%v", f, err)
				http.Error(w, "failed to add Dir to zip", http.StatusInternalServerError)
				return
			}
		}
	}
	defer zw.Close()
}

func addFileToZip(zw *zip.Writer, filePath string) error {

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
	// берём весь путь и отделяем от него конечный файл для сохранения в архив
	filename := filepath.Base(filePath)
	h.Name = strings.ReplaceAll(filename, "\\", "/")
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
		ap := strings.ReplaceAll(filepath.Join(dirPath, e.Name()), "\\", "/")
		if e.IsDir() {
			if err := addDirToZip(zw, fullFilePath, ap); err != nil {
				return err
			}
		} else {
			if err := addFileToZip(zw, fullFilePath); err != nil {
				return err
			}
		}
	}
	return nil
}

// findFiles ищет файлы в директории pathLogs, которые содержат nameFile в названии
// и имеют время изменения соответствующее dateLog
func findFiles(dateLog string, pathLogs string, nameFile string) ([]string, error) {
	var foundFiles []string

	// Парсим дату из строки
	// Формат: "2006-01-02" (можно адаптировать)
	targetDate, err := time.Parse("2006-01-02", dateLog)
	if err != nil {
		return nil, fmt.Errorf("ошибка парсинга даты: %w", err)
	}

	// Получаем конец дня (23:59:59) для сравнения
	nextDay := targetDate.AddDate(0, 0, 1)

	// Ходим по директории
	err = filepath.Walk(pathLogs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Пропускаем директории
		if info.IsDir() {
			return nil
		}

		// Проверяем наличие nameFile в названии файла
		if nameFile != "" && !contains(info.Name(), nameFile) {
			return nil
		}

		// Проверяем дату изменения файла
		modTime := info.ModTime()
		if modTime.After(targetDate) && modTime.Before(nextDay) {
			foundFiles = append(foundFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("ошибка при обходе директории: %w", err)
	}

	if len(foundFiles) == 0 {
		return nil, fmt.Errorf("файлы не найдены")
	}

	return foundFiles, nil
}

// contains проверяет, содержит ли строка haystack подстроку needle
func contains(haystack, needle string) bool {
	return len(needle) > 0 && (needle == "" || stringContains(haystack, needle))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
