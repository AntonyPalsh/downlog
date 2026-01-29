package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	// "io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Config структура конфигурации
type Config struct {
	// Port           string
	LimitMB        int64
	ApiPrefix      string
	PathLogScaners string
	PathLogTomcat  string
	ListenAddr     string
	TLSCert        string
	TLSKey         string
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
	ScanID    string `json:"scanid"`    // ID запуска сканера
}

// Получаем значение по умолчанию, если не заданны переменные окружения
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		log.Printf("🏷️ ENV %s : %v", key, value) // выводим найденное значение
		return value
	}
	log.Printf("🏷️ ENV %s : %v (default)", key, defaultValue) // выводим default, если не найдено
	return defaultValue
}

var cfg Config

func init() {

	log.Printf("🏷️ Переменные окружения:")

	// Проверяем корректность ввода значения limitMB
	limitMB, err := strconv.ParseInt(getEnv("UPT_LIMIT_DOWNLOAD_MB", "500"), 10, 64)
	if err != nil {
		log.Fatalf("❌ Не корректный формат UPT_LIMIT_DOWNLOAD_MB: %v", err)
		return
	}

	cfg = Config{
		LimitMB: limitMB,
		// Port:           getEnv("DL_PORT", ":8080"),
		ApiPrefix:      getEnv("DL_URL_API_PREFIX", ""),
		PathLogScaners: getEnv("DL_SCAN_LOG", "/app/edm/scan/logs"),
		PathLogTomcat:  getEnv("DL_TOMCAT", "/app/edm/tomcat-9/logs"),
		ListenAddr:     getEnv("DL_LISTEN_ADDR", "localhost:8080"),
		TLSCert:        getEnv("DL_CERT", "/certs/cert.crt"),
		TLSKey:         getEnv("DL_KEY", "/certs/privet.key"),
		// ApiPrefix:      getEnv("DL_URL_API_PREFIX", ""),
		// PathLogScaners: getEnv("DL_SCAN_LOG", "/home/li/code/downlog"),
		// PathLogTomcat:  getEnv("DL_TOMCAT", "/home/li/code/downlog"),
		// ListenAddr:     getEnv("DL_LISTEN_ADDR", "localhost:8080"),
		// TLSCert:        getEnv("DL_CERT", "cert.crt"),
		// TLSKey:         getEnv("DL_KEY", "privet.key"),
	}
}

// registerRoute регистрирует обработчик и сразу выводит итоговый путь в лог
func registerRoute(pattern string, handler http.HandlerFunc) {
	http.HandleFunc(pattern, handler)
	log.Printf("🔖 EdnPoint зарегистрирован: %s", pattern)
}

func main() {

	log.Printf("🔖 EdnPoints:")
	registerRoute(cfg.ApiPrefix+"/api/catalina", catalinalog)
	registerRoute(cfg.ApiPrefix+"/api/universe", universelog)
	registerRoute(cfg.ApiPrefix+"/api/scaners", scanerslog)

	// Запуск HTTP сервера
	log.Printf("🚀 Сервер запущен на https://%s", cfg.ListenAddr)

	if err := http.ListenAndServeTLS(cfg.ListenAddr, cfg.TLSCert, cfg.TLSKey, nil); err != nil {
		log.Fatalf("❌ Ошибка запуска сервера: %v", err)
	}
	// if err := http.ListenAndServe(cfg.ListenAddr, nil); err != nil {
	// 	log.Fatalf("❌ Ошибка запуска сервера: %v", err)
	// }
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
		return "", fmt.Errorf("❌ method not allowed")
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
		return "", fmt.Errorf("❌ invalid_json Invalid JSON body")
	}

	// 3) если после первого JSON-объекта в body ещё мусор/второй объект — считаем это ошибкой формата.
	if dec.More() {
		http.Error(w, "invalid_json Unexpected extra JSON content", http.StatusBadRequest)
		return "", fmt.Errorf("❌ invalid_json Unexpected extra JSON content")
	}

	if in.ScanID != "" {
		return in.ScanID, nil
	}

	s, err := parseRFC3339(in.Timestamp)
	if err != nil {
		http.Error(w, "invalid_timestamp imestamp must be RFC3339, e.g. 2026-01-23T11:07:00+03:00", http.StatusBadRequest)
		return "", fmt.Errorf("❌ invalid_timestamp imestamp must be RFC3339")
	}
	// преобразуем TimeStamp к виду в котором сохраняет файловая система и отправляем на выввод функции
	return s.Format("2006-01-02"), nil
}

// ================= EndPoints ============================================================

func catalinalog(w http.ResponseWriter, r *http.Request) {

	log.Printf("⚙️  Вызов endpoin /api/catalina")

	ts, err := validationReguest(w, r)
	if err != nil {
		log.Printf("🪠 Ошибка валидации JSON: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("🪤 Timestamp: %v", ts)

	files, err := findFiles(ts, cfg.PathLogTomcat, "catalina")
	if err != nil {
		fmt.Println("🪠 Ошибка:", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	fmt.Println("🧾 Найденные файлы:")
	for _, f := range files {
		fmt.Println(f)
	}

	handleDownload(w, files, "file")
}

// ===================================================================================
func universelog(w http.ResponseWriter, r *http.Request) {

	log.Printf("⚙️ Вызов endpoin /api/univers")

	ts, err := validationReguest(w, r)
	if err != nil {
		log.Printf("🪠 Ошибка валидации JSON: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("🪤 Timestamp: %v", ts)

	files, err := findFiles(ts, cfg.PathLogTomcat, "universe_backend")
	if err != nil {
		fmt.Println("🪠 Ошибка:", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	fmt.Println("🧾 Найденные файлы:")
	for _, f := range files {
		fmt.Println(f)
	}

	handleDownload(w, files, "file")

}

// ===================================================================================
func scanerslog(w http.ResponseWriter, r *http.Request) {

	log.Printf("⚙️ Вызов endpoin /api/scaners")

	scanID, err := validationReguest(w, r)
	if err != nil {
		log.Printf("🪠 Ошибка валидации JSON: %s", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Printf("🪤 Scaner ID: %v", scanID)

	handleDownload(w, []string{cfg.PathLogScaners + scanID}, "dir")
	// handleDownload(w, []string{"/home/li/" + scanID}, "dir")
}

//===================================================================================

// Передаём в handleDownload заголовок, список путей к файлам или папку и тип чего мы передаём "file" или "dir"
// ═══════════════════════════════════════════════════════════════════════════════
// Исправленная handleDownload - закрывает архив ДО возврата
// ═══════════════════════════════════════════════════════════════════════════════
func handleDownload(w http.ResponseWriter, files []string, typef string) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"files.zip\"")

	zw := zip.NewWriter(w)

	for _, f := range files {
		switch typef {
		case "file":
			if err := addFileToZip(zw, f); err != nil {
				log.Printf("🧾 addFileToZip failed: path=%q err=%v", f, err)
				http.Error(w, "failed to add file to zip", http.StatusInternalServerError)
				zw.Close() // ← ВАЖНО: закрыть перед возвратом!
				return
			}
		case "dir":
			if err := addDirToZip(zw, f); err != nil {
				log.Printf("📂 addDirToZip failed: path=%q err=%v", f, err)
				http.Error(w, "failed to add Dir to zip", http.StatusInternalServerError)
				zw.Close() // ← ВАЖНО: закрыть перед возвратом!
				return
			}
		}
	}

	zw.Close() // ← Закрываем архив ДО конца функции
}

// ═══════════════════════════════════════════════════════════════════════════════
// Исправленная addFileToZip - принимает только filePath
// ═══════════════════════════════════════════════════════════════════════════════
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

	// ← Используем ТОЛЬКО имя файла в архиве
	h.Name = filepath.Base(filePath)
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

// ═══════════════════════════════════════════════════════════════════════════════
// Исправленная addDirToZip - рекурсивно обходит директорию
// ═══════════════════════════════════════════════════════════════════════════════
func addDirToZip(zw *zip.Writer, dirPath string) error {
	// baseDir - это родительская директория, чтобы сохранить структуру в архиве
	baseDir := filepath.Dir(dirPath)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, e := range entries {
		fullFilePath := filepath.Join(dirPath, e.Name())

		if e.IsDir() {
			// Рекурсивно обходим вложенные директории
			if err := addDirToZip(zw, fullFilePath); err != nil {
				return err
			}
		} else {
			// Добавляем файл с сохранением относительного пути
			if err := addFileToZipWithBase(zw, fullFilePath, baseDir); err != nil {
				return err
			}
		}
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Новая функция для добавления файла с сохранением структуры директорий
// ═══════════════════════════════════════════════════════════════════════════════
func addFileToZipWithBase(zw *zip.Writer, filePath string, baseDir string) error {
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

	// Вычисляем относительный путь от baseDir
	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return err
	}

	// Преобразуем в forward slashes для архива
	h.Name = filepath.ToSlash(relPath)
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
		return nil, fmt.Errorf("файлы не найдены: %s , по пути: %s", nameFile, pathLogs)
	}

	return foundFiles, nil
}

// contains проверяет, содержит ли строка haystack подстроку needle
func contains(haystack, needle string) bool {
	if needle == "" {
		return true // пустой фильтр = все файлы
	}
	return stringContains(haystack, needle)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// findDirs ищет директории, дата модификации которых совпадает с dateLog
// dateLog ожидается в формате "2006-01-02" (ISO 8601 / YYYY-MM-DD)
// Возвращает срез путей директорий и ошибку, если она произошла
// func findDirs(dateLog string, pathLogs string) ([]string, error) {

// 	// Парсим целевую дату в формате YYYY-MM-DD
// 	targetDate, err := time.Parse("2006-01-02", dateLog)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid date format: %w", err)
// 	}

// 	var result []string

// 	// WalkDir - эффективный способ обхода директорий (Go 1.16+)
// 	err = filepath.WalkDir(pathLogs, func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			// Логируем ошибку доступа, но продолжаем обход
// 			return nil
// 		}

// 		// Проверяем только директории (исключаем файлы)
// 		if !d.IsDir() {
// 			return nil
// 		}

// 		// Получаем информацию о файле для доступа к времени модификации
// 		info, err := d.Info()
// 		if err != nil {
// 			return nil
// 		}

// 		// Сравниваем дату модификации с целевой датой
// 		// Преобразуем обе даты в полночь для сравнения только по дате
// 		modTime := info.ModTime()
// 		modDate := time.Date(modTime.Year(), modTime.Month(), modTime.Day(), 0, 0, 0, 0, time.UTC)

// 		if modDate.Equal(targetDate) {
// 			result = append(result, path)
// 		}

// 		return nil
// 	})

// 	if err != nil {
// 		return nil, fmt.Errorf("error walking directory: %w", err)
// 	}

// 	return result, nil
// }
