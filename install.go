// +build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

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
