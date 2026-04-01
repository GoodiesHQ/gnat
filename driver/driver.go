package driver

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/goodieshq/gnat/device"
	"github.com/goodieshq/gnat/utils"
)

// Profile defines the minimal configuration that differentiates one driver from another.
type Profile struct {
	// UserPrompt matches the user-mode prompt. nil for devices that have no user exec mode
	UserPrompt *regexp.Regexp

	// PrivPrompt matches the privileged-mode prompt. Required.
	PrivPrompt *regexp.Regexp

	// EnableCmd is the command to enter privileged mode.
	// Only executed if UserPrompt is non-nil and the initial prompt matched it.
	EnableCmd string

	// PasswordPrompt matches the enable-password prompt (e.g. "Password: ").
	// When non-nil and matched after sending EnableCmd, the value of
	// Settings.EnablePassword is sent as the response.
	PasswordPrompt *regexp.Regexp

	// DisablePaging is the command to disable paginated output.
	// Leave empty if the device does not paginate.
	DisablePaging string

	// Errors contains patterns that indicate a device-side error in command output
	Errors []*regexp.Regexp
}

// Driver implements device.Device for any switch described by a Profile.
type Driver struct {
	device.Settings
	Profile Profile
}

func (d *Driver) defaultTimeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return device.DefaultTimeout
}

// allPrompts returns the active set of prompt patterns: always PrivPrompt, plus
// UserPrompt if the device has one.
func (d *Driver) allPrompts() []*regexp.Regexp {
	if d.Profile.UserPrompt != nil {
		return []*regexp.Regexp{d.Profile.PrivPrompt, d.Profile.UserPrompt}
	}
	return []*regexp.Regexp{d.Profile.PrivPrompt}
}

// readUntilPrompt reads until a privileged or user prompt appears.
func (d *Driver) readUntilPrompt(ctx context.Context, timeout time.Duration) ([]byte, error) {
	prompts := d.allPrompts()
	return d.Connection.ReadUntilFunc(ctx, timeout, func(b []byte) bool {
		for _, p := range prompts {
			if p.Match(b) {
				return true
			}
		}
		return false
	})
}

// atUserPrompt returns true if the normalized raw bytes end with the user-exec prompt.
// Used by Initialize to decide whether enable is needed.
func (d *Driver) atUserPrompt(raw []byte) bool {
	if d.Profile.UserPrompt == nil {
		return false
	}
	return d.Profile.UserPrompt.Match(utils.NormalizeLineEndings(utils.StripANSI(raw)))
}

// Cmd sends a command and returns sanitized output with error detection.
// Pass timeout=0 to use the configured default.
func (d *Driver) Cmd(ctx context.Context, timeout time.Duration, command string) (*device.Result, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if timeout == 0 {
		timeout = d.defaultTimeout()
	}
	if err := d.Connection.Send([]byte(command + "\n")); err != nil {
		return nil, fmt.Errorf("send %q: %w", command, err)
	}
	raw, err := d.readUntilPrompt(ctx, timeout)
	if err != nil {
		return nil, fmt.Errorf("read after %q: %w", command, err)
	}
	output := strings.TrimSpace(strings.TrimPrefix(utils.Sanitize(raw, d.allPrompts()), command))

	result := &device.Result{Command: command, Output: output}
	for _, pat := range d.Profile.Errors {
		if m := pat.FindString(output); m != "" {
			result.Failed = true
			result.FailMsg = strings.TrimSpace(m)
			return result, nil
		}
	}
	return result, nil
}

// readUntilPrivPrompt reads until the privileged-mode prompt appears.
func (d *Driver) readUntilPrivPrompt(ctx context.Context, timeout time.Duration) ([]byte, error) {
	return d.Connection.ReadUntilFunc(ctx, timeout, func(b []byte) bool {
		return d.Profile.PrivPrompt.Match(b)
	})
}

// Initialize wakes the device, auto-detects whether enable is needed, then
// disables paging. Enable is only run if UserPrompt is set and the device
// responded with the user-exec prompt — if it came up in privileged mode
// already, the enable step is skipped automatically.
//
// If the device prompts for a password after the enable command, and
// Settings.EnablePassword is set, it is sent automatically. If
// EnablePassword is nil an empty string is sent (covers devices that accept
// a blank enable password).
func (d *Driver) Initialize(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.Connection.Send([]byte("\n")); err != nil {
		return fmt.Errorf("initialize wake: %w", err)
	}
	raw, err := d.readUntilPrompt(ctx, d.defaultTimeout())
	if err != nil {
		return fmt.Errorf("initialize prompt: %w", err)
	}
	if d.Profile.EnableCmd != "" && d.atUserPrompt(raw) {
		if err := d.Connection.Send([]byte(d.Profile.EnableCmd + "\n")); err != nil {
			return fmt.Errorf("enable send: %w", err)
		}
		// Read until either the priv prompt or a password prompt.
		raw, err = d.Connection.ReadUntilFunc(ctx, d.defaultTimeout(), func(b []byte) bool {
			return d.Profile.PrivPrompt.Match(b) ||
				(d.Profile.PasswordPrompt != nil && d.Profile.PasswordPrompt.Match(b))
		})
		if err != nil {
			return fmt.Errorf("enable: %w", err)
		}
		// If we stopped at a password prompt, send the enable password.
		normalized := utils.NormalizeLineEndings(utils.StripANSI(raw))
		if d.Profile.PasswordPrompt != nil && d.Profile.PasswordPrompt.Match(normalized) {
			pwd := ""
			if d.Settings.EnablePassword != nil {
				pwd = *d.Settings.EnablePassword
			}
			if err := d.Connection.Send([]byte(pwd + "\n")); err != nil {
				return fmt.Errorf("enable password send: %w", err)
			}
			if _, err = d.readUntilPrivPrompt(ctx, d.defaultTimeout()); err != nil {
				return fmt.Errorf("enable after password: %w", err)
			}
		}
	}
	return d.DisablePaging(ctx)
}

// DisablePaging runs the paging-disable command if one is configured.
func (d *Driver) DisablePaging(ctx context.Context) error {
	if d.Profile.DisablePaging == "" {
		return nil
	}
	result, err := d.Cmd(ctx, 0, d.Profile.DisablePaging)
	if err != nil {
		return fmt.Errorf("disable paging: %w", err)
	}
	if result.Failed {
		return fmt.Errorf("disable paging: %s", result.FailMsg)
	}
	return nil
}

// ─── Registry ────────────────────────────────────────────────────────────────

// Factory creates a device.Device from a Settings value.
type Factory func(device.Settings) device.Device

var registry = make(map[string]Factory)

// Register adds a named driver factory. Panics if the name is already registered
// (same contract as http.Handle — registration happens at init time).
func Register(name string, f Factory) {
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("drivers: %q already registered", name))
	}
	registry[name] = f
}

// DriverExists reports whether a driver with the given name is registered.
func DriverExists(name string) bool {
	_, exists := registry[name]
	return exists
}

// NewDevice creates a device.Device using the named driver factory.
// Pass a non-nil enablePassword if the device requires a password after the
// enable command; pass nil if no enable password is needed.
func NewDevice(name string, connection device.Connection, timeout time.Duration, enablePassword *string) (device.Device, error) {
	f, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("drivers: %q not found", name)
	}
	return f(device.Settings{
		DriverName:     name,
		Connection:     connection,
		Timeout:        timeout,
		EnablePassword: enablePassword,
	}), nil
}
