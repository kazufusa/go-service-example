// +build windows

package main

import (
	"os"
	"syscall"
	"time"
)

var (
	beepFunc = syscall.MustLoadDLL("user32.dll").MustFindProc("MessageBeep")
)

func beep() {
	beepFunc.Call(0xffffffff)
	now := time.Now()
	if now.Second() == now.Truncate(10*time.Second).Second() {
		os.Exit(100)
	}
}
