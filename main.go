// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop, pause or continue.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

type Service struct {
	name    string
	logger  *log.Logger
	feature IFeature
}

var IsIntSess = true

func init() {
	var err error
	IsIntSess, err = svc.IsAnInteractiveSession()

	if err != nil {
		panic(err)
	}
}

type EventLogWriter struct {
	el  *eventlog.Log
	eId uint32
}

func NewEventLogWriter(svcname string, eId uint32) (*EventLogWriter, error) {
	el, err := eventlog.Open(svcname)
	if err != nil {
		return nil, err
	}
	return &EventLogWriter{el: el, eId: eId}, nil
}

func (elw *EventLogWriter) Close() error {
	return elw.el.Close()
}

func (elw *EventLogWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	if strings.Contains(s, "[Info]") {
		err = elw.el.Info(elw.eId, s)
	} else if strings.Contains(s, "[Warn]") {
		err = elw.el.Warning(elw.eId, s)
	} else if strings.Contains(s, "[Error]") {
		err = elw.el.Error(elw.eId, s)
	}
	return 0, err
}

func (s *Service) Run() {}

func (s *Service) RunInInteractive() {}

func (s *Service) Install() {}

func (s *Service) Uninstall() {}

func (s *Service) Start() {}

func (s *Service) Stop() {}

func (s *Service) Pause() {}

func (s *Service) Continue() {}

func main() {
	const svcName = "myservice"

	// is service?
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !isIntSess {
		// is service
		runService(svcName, false)
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}
	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		runService(svcName, true)
		return
	case "install":
		err = installService(svcName, "my service")
	case "remove":
		err = removeService(svcName)
	case "start":
		err = startService(svcName)
	case "stop":
		err = controlService(svcName, svc.Stop, svc.Stopped)
	case "pause":
		err = controlService(svcName, svc.Pause, svc.Paused)
	case "continue":
		err = controlService(svcName, svc.Continue, svc.Running)
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
