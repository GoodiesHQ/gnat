package device

import (
	"context"
	"time"
)

const DefaultTimeout = 30 * time.Second

// Device is the minimal interface for interacting with a network switch.
type Device interface {
	Initialize(ctx context.Context) error
	DisablePaging(ctx context.Context) error
	Cmd(ctx context.Context, timeout time.Duration, command string) (*Result, error)
}

// InputCondition returns true when the accumulated output satisfies a completion criterion.
type InputCondition func([]byte) bool

// Connection is the low-level I/O interface for a device session.
type Connection interface {
	Start(ctx context.Context) error
	Stop() error
	Send(data []byte) error
	ReadUntilFunc(ctx context.Context, timeout time.Duration, f InputCondition) ([]byte, error)
}

// Settings holds the connection and timing configuration shared by all drivers.
type Settings struct {
	DriverName     string
	Connection     Connection
	Timeout        time.Duration // default command timeout; 0 uses DefaultTimeout
	EnablePassword *string       // nil means no enable password is required
}

// Result is the output of a single command execution.
type Result struct {
	Command string
	Output  string
	Failed  bool
	FailMsg string
}
