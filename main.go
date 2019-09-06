// +build windows

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
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

type service struct {
	feature IFeature
	logger  *log.Logger
}

func (s *service) Execute(
	args []string,
	r <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
) (ssec bool, errno uint32) {
	s.logger.Printf("[Info] argument is : %s", strings.Join(args, "-"))

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}

	s.feature.Start()
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				s.feature.Shutdown(ctx)
				break loop
			case svc.Pause:
				// changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				s.logger.Println("[Warn] this application is not pausable and restartable")
			case svc.Continue:
				// changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				s.logger.Println("[Warn] this application is not pausable and restartable")
			default:
				s.logger.Printf("[Error] unexpected control request #%d\n", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

var _ svc.Handler = (*service)(nil)

type ServiceManager struct {
	Name    string
	Desc    string
	Logger  *log.Logger
	eId     uint32
	service *service
}

var isIntSess = true

func init() {
	var err error
	isIntSess, err = svc.IsAnInteractiveSession()

	if err != nil {
		panic(err)
	}
}

func IsIntSess() bool {
	return isIntSess
}

func (sm *ServiceManager) run(isDebug bool) {
	sm.Logger.Printf("[Info] starting %s service", sm.Name)
	run := debug.Run
	if !isDebug {
		run = svc.Run
	}
	w1, err := NewEventLogWriter(sm.Name, sm.eId)
	if err != nil {
		sm.Logger.Printf("[Error] %s service logger initialization failed: %v", sm.Name, err)
		return
	}
	defer w1.Close()
	log.SetOutput(io.MultiWriter(w1, defaultLogWriter()))
	err = run(sm.Name, sm.service)
	if err != nil {
		sm.Logger.Printf("[Error] %s service failed: %v", sm.Name, err)
		return
	}
	sm.Logger.Printf("[Info] %s service stopped", sm.Name)
}

func (sm *ServiceManager) Run() {
	sm.run(false)
}

func (sm *ServiceManager) RunInInteractive() {
	sm.run(true)
}

func (sm *ServiceManager) Install() error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(sm.Name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", sm.Name)
	}
	s, err = m.CreateService(sm.Name, exepath, mgr.Config{DisplayName: sm.Desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(sm.Name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}

	// open port on firewall
	for _, inout := range []string{"in"} {
		err = exec.Command(
			"netsh",
			"advfirewall",
			"firewall",
			"add",
			"rule",
			fmt.Sprintf("name=\"%s-%s\"", sm.Name, inout),
			fmt.Sprintf("dir=%s", inout),
			"action=allow",
			fmt.Sprintf("program=\"%s\"", exepath),
			"protocol=TCP",
			"localport=80",
			"enable=yes",
		).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *ServiceManager) Uninstall() error {

	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(sm.Name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", sm.Name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(sm.Name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}

	// close port on firewall
	for _, inout := range []string{"in"} {
		err = exec.Command(
			"netsh",
			"advfirewall",
			"firewall",
			"delete",
			"rule",
			fmt.Sprintf("name=\"%s-%s\"", sm.Name, inout),
		).Run()
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *ServiceManager) Start() error {
	return startService(sm.Name)
}

func (sm *ServiceManager) Stop() error {
	return controlService(sm.Name, svc.Stop, svc.Stopped)
}

func (sm *ServiceManager) Pause() error {
	return controlService(sm.Name, svc.Pause, svc.Paused)
}

func (sm *ServiceManager) Continue() error {
	return controlService(sm.Name, svc.Continue, svc.Running)
}

func main() {
	var err error
	const (
		svcName = "myservice"
		svcDesc = "my service"
		eId     = 1
	)

	logger := log.New(defaultLogWriter(), "", log.LstdFlags|log.Lshortfile)
	sm := ServiceManager{
		Name:   svcName,
		Desc:   svcDesc,
		Logger: logger,
		eId:    eId,
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
