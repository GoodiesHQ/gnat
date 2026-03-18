package device

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/goodieshq/gnat/utils"
	"github.com/rs/zerolog/log"
)

// NewConnection creates a Connection from an SSH stdin/stdout pair.
func NewConnection(stdin io.WriteCloser, stdout io.Reader) Connection {
	return &conn{
		stdin:  stdin,
		stdout: stdout,
	}
}

type conn struct {
	stdin  io.WriteCloser
	stdout io.Reader
	ch     chan []byte  // owned by the reader goroutine
	mu     sync.RWMutex // guards ch and cancel
	cancel context.CancelFunc
}

// Start launches the background reader goroutine. Safe to call multiple times —
// subsequent calls are no-ops if the reader is already running.
func (c *conn) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.ch != nil {
		return nil // already running
	}

	c.ch = make(chan []byte)
	ctxInternal, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go func() {
		defer func() {
			c.mu.Lock()
			close(c.ch)
			c.ch = nil
			c.mu.Unlock()
			log.Debug().Msg("device reader stopped")
		}()

		log.Debug().Msg("device reader started")
		buf := make([]byte, 4096)

		// Continuously read from the device until an error occurs (e.g. session closed).
		// Feed each chunk into c.ch
		for {
			n, err := c.stdout.Read(buf)
			if err != nil {
				if !errors.Is(err, io.EOF) {
					log.Error().Err(err).Msg("device read error")
				}
				return
			}
			chunk := make([]byte, n)
			copy(chunk, buf[:n])
			select {
			case <-ctxInternal.Done():
				return
			case c.ch <- chunk:
			}
		}
	}()

	return nil
}

// Stop signals the reader goroutine to exit. The goroutine may remain alive until
// the underlying SSH session closes (blocking Read returns).
func (c *conn) Stop() error {
	c.mu.Lock()
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// Send writes raw bytes to the device.
func (c *conn) Send(data []byte) error {
	_, err := c.stdin.Write(data)
	return err
}

// ReadUntilFunc reads from the device until the provided InputCondition returns true
// Returns the sanitized output read up to and including the match that caused f to return true (or an error)
func (c *conn) ReadUntilFunc(ctx context.Context, timeout time.Duration, f InputCondition) ([]byte, error) {
	if err := c.Start(ctx); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var buf bytes.Buffer
	for {
		select {
		case <-ctx.Done():
			return buf.Bytes(), fmt.Errorf("context cancelled")
		case <-timeoutCtx.Done():
			return buf.Bytes(), fmt.Errorf("timeout after %s without prompt match", timeout)
		case chunk, ok := <-c.ch:
			if !ok {
				return nil, fmt.Errorf("connection closed while reading")
			}
			buf.Write(chunk)
			if f(utils.NormalizeLineEndings(utils.StripANSI(buf.Bytes()))) {
				return buf.Bytes(), nil
			}
		}
	}
}
