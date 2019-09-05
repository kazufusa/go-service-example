package llog

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const (
	EventId = 1
)

type Logger struct {
	lg   *log.Logger
	wslg IWinServiceLogger
	mu   sync.Mutex
	mlv  Level
}

func NewLogger() *Logger {
	return &Logger{
		lg:  log.New(os.Stderr, "", log.LstdFlags),
		mlv: DEBUG,
	}
}

func (l *Logger) SetWinServiceLogger(wslg IWinServiceLogger) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.wslg = wslg
}

func (l *Logger) SetOutput(w io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lg.SetOutput(w)
}

func (l *Logger) SetFlags(flag int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lg.SetFlags(flag)
}

func (l *Logger) SetPrefix(prefix string) {
	l.lg.SetPrefix(prefix)
}

func (l *Logger) SetMinLevel(lv Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.mlv = lv
}

func (l *Logger) output(lv Level, s string) error {
	var err error

	if lv >= l.mlv {
		err = l.lg.Output(3, fmt.Sprintf("[%v] %s", lv, s))

		if l.wslg != nil {
			switch lv {
			case INFO:
				err = l.wslg.Info(EventId, s)
			case WARN:
				err = l.wslg.Warning(EventId, s)
			case FATAL:
				err = l.wslg.Error(EventId, s)
			}
		}

	}
	return err
}

func (l *Logger) Debug(s interface{}) {
	l.output(DEBUG, fmt.Sprint(s))
}

func (l *Logger) Info(s interface{}) {
	l.output(INFO, fmt.Sprint(s))
}

func (l *Logger) Warn(s interface{}) {
	l.output(WARN, fmt.Sprint(s))
}

func (l *Logger) Error(s interface{}) {
	l.output(ERROR, fmt.Sprint(s))
}

func (l *Logger) FATAL(s interface{}) {
	l.output(FATAL, fmt.Sprint(s))
}
