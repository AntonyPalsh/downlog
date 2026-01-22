package main

import (
	"archive/zip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const baseDir = "./files"
const port = ":8080"

func main() {

	// обработать ошибку существование директории
	_ = os.MkdirAll(baseDir, 0755)

	http.HandleFunc("/api/download", handleDownload)

	// Запуск HTTP сервера
	log.Printf("🚀 Сервер запущен на http://localhost:%s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("❌ Ошибка запуска сервера: %v", err)
	}

}

func handleDownload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Files []struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request decode json", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=download.zip")

	zw := zip.NewWriter(w)
	defer zw.Close()

	base := filepath.Clean(baseDir)
	// debug
	// log.Printf("baseDir: %v", baseDir)
	// log.Printf("base: %v", base)

	for _, f := range req.Files {
		// проверка не пришёл ли путь выходящий за директорию с логами
		if !filepath.IsLocal(f.Path) {
			log.Printf("path escapes baseDir: path=%q ", f.Path)
			http.Error(w, "path escapes baseDir:", http.StatusInternalServerError)
			return
		}

		fullFilePath := filepath.Clean(filepath.Join(baseDir, f.Path))

		// debug
		// log.Printf("f.Path: %v", f.Path)
		// log.Printf("fullFilePath: %v", fullFilePath)

		if !strings.HasPrefix(fullFilePath, base) {
			continue
		}
		switch f.Type {
		case "file":
			if err := addFileToZip(zw, fullFilePath, f.Path); err != nil {
				log.Printf("addFileToZip failed: path=%q err=%v", f.Path, err)
				http.Error(w, "failed to add file to zip", http.StatusInternalServerError)
				return
			}
		case "directory":
			if err := addDirToZip(zw, fullFilePath, f.Path); err != nil {
				log.Printf("addFileToZip failed: path=%q err=%v", f.Path, err)
				http.Error(w, "failed to add Dir to zip", http.StatusInternalServerError)
				return
			}
		}
	}
}

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
