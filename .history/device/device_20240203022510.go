package device

import (
	"context"
	"regexp"
	"time"
)

type Device interface {
	Sanitize([]byte) string                       // sanitizes output from switches
	RegexInit() *regexp.Regexp                    // identifies the start of a connection
	RegexCmd() *regexp.Regexp                     // identifies the start of the next prompt and the output of the current command
	Initialize(context.Context) error             // run any commands necessary to start the connection
	DisablePaging(context.Context) error          // stop the switch from taking breaks in between long outputs
	Cmd(context.Context, string) (*Result, error) // Run a command and receive the output/switch error as a result
}

type DeviceInputCondition func([]byte) bool

type DeviceConnection interface {
	Start(context.Context) error
	Stop() error
	ReadUntilFunc(context.Context, time.Duration, DeviceInputCondition) ([]byte, error)
	ReadUntilMatch(context.Context, time.Duration, *regexp.Regexp) ([]byte, error)
	ReadFor(context.Context, time.Duration) ([]byte, error)
	Send([]byte) error
}

type DeviceSettings struct {
	Conection    DeviceConnection
	TimeoutRead  time.Duration
	TimeoutWrite time.Duration
}
