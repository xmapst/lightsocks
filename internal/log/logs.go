package log

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/xmapst/lightsocks/internal/observable"
	"path"
	"strings"
)

var (
	logCh  = make(chan any)
	source = observable.NewObservable(logCh)
)

type Event struct {
	LogLevel logrus.Level
	Payload  string
}

func (e *Event) Type() string {
	return e.LogLevel.String()
}

func Subscribe() observable.Subscription {
	sub, _ := source.Subscribe()
	return sub
}

func UnSubscribe(sub observable.Subscription) {
	source.UnSubscribe(sub)
}

func newLogln(logLevel logrus.Level, v ...any) Event {
	msg := fmt.Sprintln(v...)
	return Event{
		LogLevel: logLevel,
		Payload:  msg[:len(msg)-1],
	}
}

func newLogf(logLevel logrus.Level, format string, v ...any) Event {
	return Event{
		LogLevel: logLevel,
		Payload:  fmt.Sprintf(format, v...),
	}
}

type ConsoleFormat struct{}

func (c *ConsoleFormat) Format(entry *logrus.Entry) ([]byte, error) {
	logCh <- newLogln(entry.Level, entry.Message)
	logStr := fmt.Sprintf("%s %s %s:%d %v\n",
		entry.Time.Format("2006/01/02 15:04:05"),
		strings.ToUpper(entry.Level.String()),
		path.Base(entry.Caller.File),
		entry.Caller.Line,
		entry.Message,
	)
	return []byte(logStr), nil
}
