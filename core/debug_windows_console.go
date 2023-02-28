//go:build windows && debug

package core

import (
	"fmt"
	"os"
	"syscall"
)

func SetupWindowsConsole() {
	modkernel32 := syscall.NewLazyDLL("kernel32.dll")
	procAllocConsole := modkernel32.NewProc("AllocConsole")
	r0, _, err0 := syscall.Syscall(procAllocConsole.Addr(), 0, 0, 0, 0)
	if r0 == 0 { // Allocation failed, probably process already has a console
		fmt.Printf("Could not allocate console: %s. Check build flags..", err0)
		os.Exit(1)
	}
	hout, err1 := syscall.GetStdHandle(syscall.STD_OUTPUT_HANDLE)
	hin, err2 := syscall.GetStdHandle(syscall.STD_INPUT_HANDLE)
	if err1 != nil || err2 != nil { // nowhere to print the error
		os.Exit(2)
	}
	os.Stdout = os.NewFile(uintptr(hout), "/dev/stdout")
	os.Stdin = os.NewFile(uintptr(hin), "/dev/stdin")
}
