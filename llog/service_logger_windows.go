// +build windows

package llog

import "golang.org/x/sys/windows/svc/eventlog"

var _ IWinServiceLogger = (*eventlog.Log)(nil)

func NewWinServiceLogger(svcname string) (IWinServiceLogger, error) {
	return eventlog.Open(svcname)
}
