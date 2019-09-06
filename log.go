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
	if strings.Contains(s, "[Info]") {
		err = elw.el.Info(elw.eId, s)
	} else if strings.Contains(s, "[Warn]") {
		err = elw.el.Warning(elw.eId, s)
	} else if strings.Contains(s, "[Error]") {
		err = elw.el.Error(elw.eId, s)
	}
	return 0, err
}

func defaultLogWriter() io.Writer {
	return io.MultiWriter(
		&lumberjack.Logger{
			Filename:   "C:/Users/user/Desktop/work/aaaa.log",
			MaxSize:    500, // megabytes
			MaxBackups: 3,
			MaxAge:     28, //days
		},
		os.Stdout,
	)
}
