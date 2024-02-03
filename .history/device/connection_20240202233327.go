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

type DeviceInputCondition func([]byte) bool

type DeviceConnection interface {
	Start(context.Context) error
	Stop() error
	ReadUntilFunc(context.Context, time.Duration, DeviceInputCondition)
	ReadUntilMatch(context.Context, time.Duration, *regexp.Regexp) ([]byte, error)
	ReadFor(context.Context, time.Duration) ([]byte, error)
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
	mu      sync.RWMutex
	running bool
}

func (conn *connection) Start(ctx context.Context) error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if conn.running {
		return nil
	}

	conn.running = true
	conn.ch = make(chan []byte)

	go func() {
		defer conn.Stop()

		log.Debug().Msg("started reader")
		buf := make([]byte, 1024)

		for {
			if err := ctx.Err(); err != nil {
				log.Error().Err(err).Msg("Conn ctx error")
				return
			}
			n, err := conn.stdout.Read(buf)
			if err != nil {
				log.Error().Err(err).Msg("Conn errored out")
				return
			}
			tmp := make([]byte, n)
			copy(tmp, buf[:n])
			select {
			case conn.ch <- tmp:
				log.Info().Msgf("Wrote %d bytes to channel", len(tmp))
			default:
				log.Error().Err(err).Msg("Can't write to channel")
				return
			}
		}
	}()

	return nil
}

func (conn *connection) Stop() error {
	conn.mu.Lock()
	defer conn.mu.Unlock()

	if !conn.running {
		log.Debug().Msg("connection already stopped")
		return nil
	}

	log.Debug().Msg("Stopping connection...")
	close(conn.ch)
	conn.running = false
	// conn.ch = nil
	return nil
}

func (conn *connection) ReadUntilFunc(ctx context.Context, timeout time.Duration, f func([]byte) bool) {
	conn.Start(ctx)

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var buffer bytes.Buffer

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return buffer.Bytes(), fmt.Errorf("parent context is done")
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

func (conn *connection) ReadUntilMatch(ctx context.Context, regex *regexp.Regexp, timeout time.Duration) ([]byte, error) {
	conn.Start(ctx)

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var buffer bytes.Buffer

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return buffer.Bytes(), fmt.Errorf("parent context is done")
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

func (conn *connection) ReadFor(ctx context.Context, timeout time.Duration) ([]byte, error) {
	conn.Start(ctx)

	conn.mu.RLock()
	defer conn.mu.RUnlock()

	var buffer bytes.Buffer

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return buffer.Bytes(), fmt.Errorf("parent context is done")
		case <-timeoutCtx.Done():
			return buffer.Bytes(), fmt.Errorf("timeout reached without matching regex")
		case tmp, ok := <-conn.ch:
			if !ok {
				err := fmt.Errorf("channel not ok while reading")
				log.Error().Err(err).Send()
				return nil, err
			}
			buffer.Write(tmp)
		}
	}
}

func (conn *connection) Send(data []byte) error {
	_, err := conn.stdin.Write(data)
	return err
}
