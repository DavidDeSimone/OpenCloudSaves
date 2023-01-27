//go:build darwin

package main

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed build/macos/macbuildsent.txt
var macSentinel string

func GetMacOsPath() string {
	if macSentinel != "" {
		exe, _ := os.Executable()
		dir := filepath.Dir(exe)
		parent := filepath.Dir(dir)

		return parent + "/Resources/rclone"
	} else {
		return "./bin/rclone"
	}
}
