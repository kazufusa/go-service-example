package llog

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var lavelMap = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

func (lv Level) String() string {
	if s, ok := lavelMap[lv]; ok {
		return s
	}
	return ""
}
