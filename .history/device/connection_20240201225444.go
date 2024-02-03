package device

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
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

func (conn *connection) ReadUntilMatch(ctx context.Context, regex *regexp.Regexp, timeout time.Duration) ([]byte, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var ch chan []byte
	var buffer bytes.Buffer

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	go func() {

	}()

	for {
		select {
		case <-timeoutCtx.Done():
			return buffer.Bytes(), fmt.Errorf("timeout reached without matching regex")
		case tmp, ok := <-conn.ch:
			if !ok {
				err := fmt.Errorf("channel not ok while reading")
				log.Error().Err(err).Send()
				return nil, err
			}
			buffer.Write(tmp)
		case <-ticker.C:
			if regex.Match(buffer.Bytes()) {
				return buffer.Bytes(), nil
			}
		}
	}
}
