//go:build windows

package main

import (
	"os"
	"path/filepath"

	"github.com/jchv/go-webview2"
)

// openUI shows DEMviewer in its own window (WebView2). Returns when the
// window is closed. Falls back to the default browser if WebView2 is missing.
func openUI(url string) bool {
	data := filepath.Join(os.Getenv("LOCALAPPDATA"), "DEMviewer")
	os.MkdirAll(data, 0755)
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  data,
		WindowOptions: webview2.WindowOptions{
			Title:  "DEMviewer",
			IconId: 1, // icon group from rsrc_windows_amd64.syso
			Width:  1320,
			Height: 900,
			Center: true,
		},
	})
	if w == nil {
		openBrowser(url)
		return false
	}
	defer w.Destroy()
	mainHwnd = uintptr(w.Window())
	w.Navigate(url)
	w.Run()
	return true
}
