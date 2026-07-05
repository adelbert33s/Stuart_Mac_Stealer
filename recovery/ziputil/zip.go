package ziputil

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const maxZipSize = 50 * 1024 * 1024 // 50 MB

// FileEntry maps a disk path to a zip member path.
type FileEntry struct {
	DiskPath string
	ZipPath  string
}

func ZipDirectory(dir string) ([]byte, error) {
	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("directory not found: %s", dir)
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	baseName := filepath.Base(dir)

	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		zipEntry := baseName + "/" + filepath.ToSlash(rel)
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		w, err := zw.Create(zipEntry)
		if err != nil {
			return nil
		}
		io.Copy(w, f) //nolint:errcheck
		return nil
	})

	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() > maxZipSize {
		return nil, fmt.Errorf("ZIP too large (%d bytes, max %d)", buf.Len(), maxZipSize)
	}
	return buf.Bytes(), nil
}

// ZipFileEntries builds one zip archive from the given entries.
func ZipFileEntries(entries []FileEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		if e.DiskPath == "" || e.ZipPath == "" {
			continue
		}
		f, err := os.Open(e.DiskPath)
		if err != nil {
			continue
		}
		w, err := zw.Create(filepath.ToSlash(e.ZipPath))
		if err != nil {
			f.Close()
			continue
		}
		_, _ = io.Copy(w, f)
		f.Close()
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("zip archive is empty")
	}
	return buf.Bytes(), nil
}

// ZipFileEntriesChunked packs entries into multiple zip archives under maxBytes each.
func ZipFileEntriesChunked(entries []FileEntry, maxBytes int) ([][]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no files to zip")
	}
	if maxBytes <= 0 {
		maxBytes = maxZipSize
	}

	var chunks [][]byte
	batch := make([]FileEntry, 0, len(entries))

	flush := func(b []FileEntry) error {
		if len(b) == 0 {
			return nil
		}
		z, err := ZipFileEntries(b)
		if err != nil {
			return err
		}
		chunks = append(chunks, z)
		return nil
	}

	for _, e := range entries {
		trial := append(append([]FileEntry{}, batch...), e)
		z, err := ZipFileEntries(trial)
		if err != nil {
			continue
		}
		if len(z) > maxBytes {
			if len(batch) > 0 {
				if err := flush(batch); err != nil {
					return nil, err
				}
				batch = []FileEntry{e}
				continue
			}
			// Single large file — still ship it in its own chunk.
			chunks = append(chunks, z)
			continue
		}
		batch = trial
	}
	if err := flush(batch); err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, fmt.Errorf("failed to build zip chunks")
	}
	return chunks, nil
}

func ZipFiles(paths []string, baseDir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, p := range paths {
		rel, err := filepath.Rel(baseDir, p)
		if err != nil {
			rel = filepath.Base(p)
		}
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		w, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			f.Close()
			continue
		}
		io.Copy(w, f)
		f.Close()
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() > maxZipSize {
		return nil, fmt.Errorf("ZIP too large (%d bytes, max %d)", buf.Len(), maxZipSize)
	}
	return buf.Bytes(), nil
}
