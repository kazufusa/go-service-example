// +build windows

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/kazufusa/go-service-example/llog"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

var elog *llog.Logger

var _ svc.Handler = (*myservice)(nil)

type myservice struct {
	feature Feature
}

func (m *myservice) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	elog.Info(strings.Join(args, "-"))

	ctx, cancel := context.WithCancel(context.Background())
	chContinue, chPause := m.feature.Start(ctx)

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				testOutput := strings.Join(args, "-")
				testOutput += fmt.Sprintf("-%d", c.Context)
				elog.Info(testOutput)
				cancel()
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				chPause <- struct{}{}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				chContinue <- struct{}{}
			default:
				elog.Error(fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, isDebug bool) {
	var err error

	elog = llog.NewLogger()
	if !isDebug {
		wslg, err := llog.NewWinServiceLogger(name)
		if err != nil {
			return
		}
		defer wslg.Close()
		elog.SetWinServiceLogger(wslg)
	}

	elog.Info(fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &myservice{feature: Feature{elog: elog}})
	if err != nil {
		elog.Error(fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(fmt.Sprintf("%s service stopped", name))
}
