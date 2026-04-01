package drivers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goodieshq/gnat/device"
	"github.com/goodieshq/gnat/driver"
)

// All providers except GetConfig and GetEventLog are served by a single
// "show system information" command. The output has a two-column layout where
// some values of interest share a line with an unrelated label, e.g.:
//
//	Up Time   : 165 days    Memory   - Total   : 683,610,112
//	CPU Util (%): 10                            Free    : 490,370,760
//
// The memory cross-line regex bridges exactly one newline with [^\n]* to reach
// the Free value on the following line without using dot-all mode.
//
// The uptime regex stops capture at two or more consecutive spaces, which
// serve as the visual column separator in the two-column layout.
var (
	procurveSysNameRe   = regexp.MustCompile(`(?m)System Name\s+:\s+(\S+)`)
	procurveSysSWRe     = regexp.MustCompile(`(?m)Software revision\s+:\s+(\S+)`)
	procurveSysROMRe    = regexp.MustCompile(`(?m)ROM Version\s+:\s+(\S+)`)
	procurveSysSerialRe = regexp.MustCompile(`(?m)Serial Number\s+:\s+(\S+)`)
	procurveSysCPURe    = regexp.MustCompile(`(?m)CPU Util \(%\)\s+:\s+(\d+)`)
	procurveSysMemRe    = regexp.MustCompile(`Memory\s+-\s+Total\s+:\s+([\d,]+)[^\n]*\n[^\n]*Free\s+:\s+([\d,]+)`)
	// Captures the uptime value before the column-separator whitespace (2+ spaces).
	procurveSysUptimeRe = regexp.MustCompile(`(?m)Up Time\s+:\s+(.+?)(?:\s{2,}|$)`)
)

type ProcurveDriver struct {
	*driver.Driver
}

// sysInfo runs "show system information" and returns the output.
func (d *ProcurveDriver) sysInfo(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "show system information")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

// stripCommas removes thousands-separator commas so "683,610,112" parses cleanly.
func stripCommas(s string) string {
	return strings.ReplaceAll(s, ",", "")
}

func (d *ProcurveDriver) GetHostname(ctx context.Context, timeout time.Duration) (string, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return "", err
	}
	m := procurveSysNameRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return "", fmt.Errorf("hostname not found in show system information")
	}
	return m[1], nil
}

func (d *ProcurveDriver) GetVersion(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return nil, err
	}
	m := procurveSysSWRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return []*driver.FirmwareInfo{}, nil
	}
	return []*driver.FirmwareInfo{{Version: m[1]}}, nil
}

func (d *ProcurveDriver) GetVersionBootROM(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return nil, err
	}
	m := procurveSysROMRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return []*driver.FirmwareInfo{}, nil
	}
	return []*driver.FirmwareInfo{{Version: m[1]}}, nil
}

func (d *ProcurveDriver) GetSerialNumber(ctx context.Context, timeout time.Duration) ([]string, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return nil, err
	}
	m := procurveSysSerialRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return nil, fmt.Errorf("serial number not found in show system information")
	}
	return []string{m[1]}, nil
}

func (d *ProcurveDriver) GetCPU(ctx context.Context, timeout time.Duration) (int, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return 0, err
	}
	m := procurveSysCPURe.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, fmt.Errorf("CPU utilization not found in show system information")
	}
	return strconv.Atoi(m[1])
}

func (d *ProcurveDriver) GetRAM(ctx context.Context, timeout time.Duration) (*driver.RAMInfo, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return nil, err
	}
	m := procurveSysMemRe.FindStringSubmatch(out)
	if len(m) < 3 {
		return nil, fmt.Errorf("memory info not found in show system information")
	}
	total, err := strconv.ParseInt(stripCommas(m[1]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse memory total: %w", err)
	}
	free, err := strconv.ParseInt(stripCommas(m[2]), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse memory free: %w", err)
	}
	return &driver.RAMInfo{Total: total, Free: free}, nil
}

func (d *ProcurveDriver) GetUptime(ctx context.Context, timeout time.Duration) (int64, error) {
	out, err := d.sysInfo(ctx, timeout)
	if err != nil {
		return 0, err
	}
	m := procurveSysUptimeRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, fmt.Errorf("uptime not found in show system information")
	}
	secs, err := driver.ParseUptimeSeconds(m[1])
	if err != nil {
		return 0, fmt.Errorf("parse uptime: %w", err)
	}
	return secs, nil
}

func (d *ProcurveDriver) GetEventLog(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "show log")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

func (d *ProcurveDriver) GetConfig(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "write terminal")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

// ProcurveProfile defines the command prompt and error patterns for HP Procurve switches.
var procurveProfile = driver.Profile{
	PrivPrompt: regexp.MustCompile(`\S+#\s*$`),
	UserPrompt: regexp.MustCompile(`\S+>\s*$`),
	Errors: []*regexp.Regexp{
		regexp.MustCompile(`(?i)invalid input`),
		regexp.MustCompile(`(?i)ambiguous command`),
		regexp.MustCompile(`(?i)error:`),
	},
	DisablePaging: "no page",
}

func init() {
	driver.Register("procurve", func(s device.Settings) device.Device {
		return &ProcurveDriver{
			Driver: &driver.Driver{Settings: s, Profile: procurveProfile},
		}
	})
}
