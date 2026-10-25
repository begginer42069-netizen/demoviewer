package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type Settings struct {
	DemoFolder string `json:"demoFolder"`
	MapsFolder string `json:"mapsFolder"`
	// cstrike\\models (player and weapon models for the replay)
	ModelsFolder string `json:"modelsFolder"`
}

var (
	settings   Settings
	settingsMu sync.Mutex
)

func settingsPath() string {
	if runtime.GOOS == "windows" {
		if la := os.Getenv("LOCALAPPDATA"); la != "" {
			return filepath.Join(la, "DEMviewer", "settings.json")
		}
	}
	return filepath.Join(exeDir(), "demviewer_settings.json")
}

func loadSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	if b, err := os.ReadFile(settingsPath()); err == nil {
		json.Unmarshal(b, &settings)
	}
}

func saveSettings() {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	p := settingsPath()
	os.MkdirAll(filepath.Dir(p), 0755)
	b, _ := json.MarshalIndent(settings, "", "  ")
	os.WriteFile(p, b, 0644)
}
