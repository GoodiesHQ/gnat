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

type conn struct {
	stdin  io.WriteCloser
	stdout io.Reader
	ch     chan []byte        // owned by the reader goroutine
	cancel context.CancelFunc // cancel func  to kill the conn's context
	err    error              // any early connection error
	mu     sync.RWMutex       // guards ch and cancel
}

// Start launches the background reader goroutine. Safe to call multiple times —
// subsequent calls are no-ops if the reader is already running.
func (c *conn) Start(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.err != nil {
		return c.err
	}

	if c.ch != nil {
		return nil // already running
	}

	c.ch = make(chan []byte)
	ctxInternal, cancel := context.WithCancel(ctx)
	c.cancel = cancel

	go func(ch chan []byte) {
		var errConn error

		defer func() {
			cancel()
			c.mu.Lock()
			if c.err == nil {
				c.err = errConn
			}
			close(ch)
			c.mu.Unlock()
			log.Debug().Msg("device reader stopped")
		}()

		log.Debug().Msg("device reader started")
		buf := make([]byte, 4096)

		// Continuously read from the device until an error occurs (e.g. session closed).
		// Feed each chunk into ch
		for {
			if err := ctxInternal.Err(); err != nil {
				errConn = err
				return
			}

			n, err := c.stdout.Read(buf)

			// Readers could return data + an error
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])

				select {
				case <-ctxInternal.Done():
					errConn = ctxInternal.Err()
					return
				case ch <- chunk:
				}
			}

			if err != nil {
				errConn = err
				if !errors.Is(err, io.EOF) {
					log.Error().Err(err).Msg("device read error")
				}
				return
			}
		}
	}(c.ch)

	return nil
}

// Stop signals the reader goroutine to exit. The goroutine may remain alive until
// the underlying SSH session closes (blocking Read returns).
func (c *conn) Stop() error {
	c.mu.Lock()
	// If no other error, mark pipe as closed
	if c.err == nil {
		c.err = io.ErrClosedPipe
	}
	cancel := c.cancel
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// Send writes raw bytes to the device.
func (c *conn) Send(data []byte) error {
	c.mu.RLock()
	err := c.err
	c.mu.RUnlock()
	if err != nil {
		return err
	}

	_, err = c.stdin.Write(data)
	return err
}

// ReadUntilFunc reads from the device until the provided InputCondition returns true
// Returns the sanitized output read up to and including the match that caused f to return true (or an error)
func (c *conn) ReadUntilFunc(ctx context.Context, timeout time.Duration, f InputCondition) ([]byte, error) {
	// check ctx err first
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	// start conn on a separate lifetime from the context
	if err := c.Start(context.Background()); err != nil {
		return nil, err
	}

	c.mu.RLock()
	ch := c.ch
	c.mu.RUnlock()

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var buf bytes.Buffer
	for {
		select {
		case <-timeoutCtx.Done():
			return buf.Bytes(), fmt.Errorf("command response read interrupted: %w", timeoutCtx.Err())
		case chunk, ok := <-ch:
			if !ok {
				c.mu.RLock()
				err := c.err
				c.mu.RUnlock()

				return buf.Bytes(), fmt.Errorf("connection closed while reading: %w", err)
			}
			buf.Write(chunk)
			if f(utils.NormalizeLineEndings(utils.StripANSI(buf.Bytes()))) {
				return buf.Bytes(), nil
			}
		}
	}
}
