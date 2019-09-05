// +build windows

package winsvc

import (
	_ "golang.org/x/sys/windows/svc/mgr"
)

// var config mgr.Config

type Svc struct {
}
