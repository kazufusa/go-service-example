// +build !windows

package llog

type mocklogger struct{}

var _ IWinServiceLogger = (*mocklogger)(nil)

func (m *mocklogger) Close() error                 { return nil }
func (m *mocklogger) Info(uint32, string) error    { return nil }
func (m *mocklogger) Warning(uint32, string) error { return nil }
func (m *mocklogger) Error(uint32, string) error   { return nil }

func NewWinServiceLogger(svcname string) (IWinServiceLogger, error) {
	return &mocklogger{}, nil
}
