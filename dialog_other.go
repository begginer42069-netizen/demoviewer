//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"strings"
)

// browseDialog on Linux/macOS uses zenity or kdialog when one is installed.
func browseDialog(kind, title, start string) (string, error) {
	if start == "" {
		start = "."
	}
	var cmd *exec.Cmd
	if p, err := exec.LookPath("zenity"); err == nil {
		args := []string{"--file-selection", "--title=" + title, "--filename=" + strings.TrimSuffix(start, "/") + "/"}
		if kind != "demo" {
			args = append(args, "--directory")
		} else {
			args = append(args, "--file-filter=Demos (*.dem) | *.dem")
		}
		cmd = exec.Command(p, args...)
	} else if p, err := exec.LookPath("kdialog"); err == nil {
		if kind != "demo" {
			cmd = exec.Command(p, "--title", title, "--getexistingdirectory", start)
		} else {
			cmd = exec.Command(p, "--title", title, "--getopenfilename", start, "*.dem")
		}
	} else {
		return "", errors.New("no file dialog found (install zenity or kdialog), or type the folder path instead")
	}
	out, err := cmd.Output()
	if err != nil {
		return "", nil // cancelled
	}
	return strings.TrimSpace(string(out)), nil
}
