//go:build !darwin

// Non-darwin stub: the Mac password GUI and Keychain flow only exist on macOS.
// Keeps the package compilable on other GOOS during development.
package main

import "errors"

func acquireMacPassword(fromFlag string, noPrompt bool, title, message string, quiet bool) (string, error) {
	_ = title
	_ = message
	_ = quiet
	_ = noPrompt
	if p := fromFlag; p != "" {
		return p, nil
	}
	return "", errors.New("GUI password prompt is only available on macOS; use -mac-password")
}