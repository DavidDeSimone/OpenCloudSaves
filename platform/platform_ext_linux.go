//go:build linux

package platform

import (
	_ "embed"
	"os"
)

var checkedLinuxPath = false
var relativeLinuxPath = false

func GetPath() string {
	if !checkedLinuxPath {
		_, err := os.Stat("./bin/rclone")
		if err != nil {
			relativeLinuxPath = false
		}
		checkedLinuxPath = true
	}

	if relativeLinuxPath {
		return "./bin/rclone"
	} else {
		return "rclone"
	}
}
