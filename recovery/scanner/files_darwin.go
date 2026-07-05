//go:build darwin

package scanner

func getScanLocations() []scanLocation {
	return []scanLocation{
		{"Desktop", "Desktop"},
		{"Documents", "Documents"},
		{"Downloads", "Downloads"},
		{"Library/Application Support", "AppSupport"},
		{"Dropbox", "Dropbox"},
	}
}