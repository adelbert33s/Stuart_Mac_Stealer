//go:build !darwin

package main

func collectLoginKeychainRawFiles() []rawDiskFile { return nil }

func collectBrowserRawDBFiles() []rawDiskFile { return nil }

type rawDiskFile struct {
	ZipPath  string
	DiskPath string
}
