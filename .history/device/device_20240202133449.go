package device

import (
	"context"
	"regexp"
)

type Device interface {
	Sanitize([]byte) string                       // sanitizes output from switches
	RegexInit() *regexp.Regexp                    // identifies the start of a connection
	RegexCmd() *regexp.Regexp                     // identifies the start of the next prompt and the output of the current command
	Initialize(context.Context) error             // run any commands necessary to start the connection
	DisablePaging(context.Context) error          // stop the switch from taking breaks in between long outputs
	Cmd(context.Context, string) (*Result, error) // Run a command and receive the output/switch error as a result
}
