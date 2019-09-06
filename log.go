package main

import (
	"io"
	"os"
	"strings"

	"golang.org/x/sys/windows/svc/eventlog"
	"gopkg.in/natefinch/lumberjack.v2"
)

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
	if strings.Contains(s, "[INFO]") {
		err = elw.el.Info(elw.eId, s)
	} else if strings.Contains(s, "[WARN]") {
		err = elw.el.Warning(elw.eId, s)
	} else if strings.Contains(s, "[ERROR]") {
		err = elw.el.Error(elw.eId, s)
	}
	return 0, err
}

type DefaultLogWriter struct {
	w           io.Writer
	shouldClose io.WriteCloser
}

func (d DefaultLogWriter) Write(p []byte) (n int, err error) {
	return d.w.Write(p)
}

func (d DefaultLogWriter) Close() error {
	return d.shouldClose.Close()
}

func NewDefaultLogWriter(path string) DefaultLogWriter {
	shouldClose := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    10, // megabytes
		MaxBackups: 3,
		MaxAge:     1500, //days
		Compress:   true,
	}

	return DefaultLogWriter{
		w:           io.MultiWriter(shouldClose, os.Stdout),
		shouldClose: shouldClose,
	}
}
