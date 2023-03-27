//go:build !windows

package platform

import "os/exec"

func SetupWindowsConsole() {

}

func StripWindow(cmd *exec.Cmd) {

}
