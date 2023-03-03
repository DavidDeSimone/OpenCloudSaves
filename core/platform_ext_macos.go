//go:build darwin

package core

import (
	_ "embed"
	"os"
	"path/filepath"
)

var checkedPath = false
var useMacRelativePath = false

func GetMacOsPath() string {
	if !checkedPath {
		_, err := os.Stat("./bin/rclone")
		if err == nil {
			useMacRelativePath = true
		}
		checkedPath = true
	}

	if useMacRelativePath {
		return "./bin/rclone"
	} else {
		exe, _ := os.Executable()
		dir := filepath.Dir(exe)
		parent := filepath.Dir(dir)

		return parent + "/Resources/rclone"
	}
}
