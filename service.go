// +build windows

package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

type service struct {
	feature IFeature
	logger  *log.Logger
}

func (s *service) Execute(
	args []string,
	r <-chan svc.ChangeRequest,
	changes chan<- svc.Status,
) (ssec bool, errno uint32) {
	s.logger.Printf("[INFO] argument is : %s", strings.Join(args, "-"))

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
				s.logger.Println("[WARN] this application is not pausable and restartable")
			case svc.Continue:
				// changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				s.logger.Println("[WARN] this application is not pausable and restartable")
			default:
				s.logger.Printf("[ERROR] unexpected control request #%d\n", c)
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

var _ svc.Handler = (*service)(nil)

type ServiceManager struct {
	Name      string
	Desc      string
	Logger    *log.Logger
	eId       uint32
	service   *service
	exePath   string
	logPath   string
	logWriter io.Writer
}

var (
	isIntSess = true
	exepath   = ""
)

func init() {
	var err error
	isIntSess, err = svc.IsAnInteractiveSession()

	if err != nil {
		panic(err)
	}

	exepath, err = exePath()
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

	log.SetOutput(io.MultiWriter(w1, sm.logWriter))
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

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

func installService(name, desc string) error {
	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}

	// open port
	// https://support.microsoft.com/ja-jp/help/947709/how-to-use-the-netsh-advfirewall-firewall-context-instead-of-the-netsh
	// may be need `s := syscall.EscapeArg(exepath)`
	for _, inout := range []string{"in"} {
		err = exec.Command(
			"netsh",
			"advfirewall",
			"firewall",
			"add",
			"rule",
			fmt.Sprintf("name=\"%s-%s\"", name, inout),
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

func removeService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}

	// close port
	for _, inout := range []string{"in"} {
		err = exec.Command(
			"netsh",
			"advfirewall",
			"firewall",
			"delete",
			"rule",
			fmt.Sprintf("name=\"%s-%s\"", name, inout),
		).Run()
		if err != nil {
			return err
		}
	}

	return nil
}

func startService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return nil
}

func controlService(name string, c svc.Cmd, to svc.State) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}
	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}
