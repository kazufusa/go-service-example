// +build windows

package main

import (
	"fmt"
	"log"
	"os"
	"strings"
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

func main() {
	var err error
	const (
		svcName = "myservice"
		svcDesc = "my service"
		eId     = 1
	)

	lw := NewDefaultLogWriter(exepath + ".log")
	defer lw.Close()

	logger := log.New(lw, "", log.LstdFlags|log.Lshortfile)
	sm := ServiceManager{
		Name:      svcName,
		Desc:      svcDesc,
		Logger:    logger,
		eId:       eId,
		exePath:   exepath,
		logPath:   exepath + ".log",
		logWriter: lw,
		service: &service{
			feature: &Feature{
				logger: logger,
			},
			logger: logger},
	}

	if !IsIntSess() {
		sm.Run()
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}
	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		sm.RunInInteractive()
		return
	case "install":
		err = sm.Install()
	case "uninstall":
		err = sm.Uninstall()
	case "start":
		err = sm.Start()
	case "stop":
		err = sm.Stop()
	case "pause":
		err = sm.Pause()
	case "continue":
		err = sm.Continue()
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcName, err)
	}
	return
}
