//go:build !darwin

package main

import "errors"

func acquireMacPassword(fromFlag string, noPrompt bool, title, message string, quiet bool) (string, error) {
	_ = title
	_ = message
	_ = quiet
	if noPrompt {
		return "", errors.New("-mac-password is required on non-macOS builds")
	}
	return "", errors.New("GUI password prompt is only available on macOS")
}