package llog

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Log_output(t *testing.T) {
	var tests = []struct {
		l Level
		m string
		e string
	}{
		{DEBUG, "Hello", "llog_test.go:36: [DEBUG] Hello\n"},
		{INFO, "Hello", "llog_test.go:36: [INFO] Hello\n"},
		{ERROR, "Hello", "llog_test.go:36: [ERROR] Hello\n"},
		{WARN, "Hello", "llog_test.go:36: [WARN] Hello\n"},
		{FATAL, "Hello", "llog_test.go:36: [FATAL] Hello\n"},
	}
	for _, tt := range tests {
		t.Run(tt.l.String(), func(t *testing.T) {
			buf := new(bytes.Buffer)
			logger := NewLogger()

			logger.SetOutput(buf)
			logger.SetFlags(log.Lshortfile)

			wslg, err := NewWinServiceLogger("testapp")
			assert.NoError(t, err)
			logger.SetWinServiceLogger(wslg)
			defer wslg.Close()

			assert.NoError(t, (func() error { return logger.output(tt.l, tt.m) })())
			assert.Equal(t, tt.e, buf.String())
		})
	}
}

func Test_Log_Info(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := NewLogger()

	logger.SetOutput(buf)
	logger.SetFlags(log.Lshortfile)

	logger.Info("hello")
	assert.Equal(t, "llog_test.go:49: [INFO] hello\n", buf.String())
}
