package device

import (
	"context"
	"io"
	"regexp"
	"sync"
	"time"
)

type NetworkSwitchConnection interface {
	Start(context.Context) error
	Stop() error
	ReadUntilMatch(context.Context, *regexp.Regexp, time.Duration) ([]byte, error)
	Send([]byte) error
}

func NewNetworkSwitchConnection(stdin io.WriteCloser, stdout io.Reader) NetworkSwitchConnection {
	return &connection{
		stdin:   stdin,
		stdout:  stdout,
		ch:      nil,
		running: false,
	}
}

type connection struct {
	stdin   io.WriteCloser // input sent to the switch
	stdout  io.Reader      // output read from the switch
	ch      chan []byte    // channel for sending chunks of bytes through
	mu      sync.Mutex
	running bool
}
