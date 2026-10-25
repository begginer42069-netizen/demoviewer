//go:build !windows

package main

func openUI(url string) bool {
	openBrowser(url)
	return false
}
