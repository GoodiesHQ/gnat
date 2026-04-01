package drivers

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/goodieshq/gnat/device"
	"github.com/goodieshq/gnat/driver"
)

var (
	// show version
	ciscoiosVersionRe   = regexp.MustCompile(`(?m)Cisco IOS Software.*Version (\S+),`)
	ciscoiosBootromRe   = regexp.MustCompile(`(?m)BOOTLDR.*Version (\S+),`)
	ciscoiosUptimeRe    = regexp.MustCompile(`(?mi)uptime is\s+(.+)`)
	ciscoiosSerialRe    = regexp.MustCompile(`(?m)System serial number\s*:\s*(\S+)`)
	ciscoiosSerialOldRe = regexp.MustCompile(`(?m)Processor board ID\s+(\S+)`) // older IOS fallback

	// show running-config | include ^hostname
	ciscoiosHostnameRe = regexp.MustCompile(`(?m)^hostname\s+(\S+)`)

	// show processes cpu | inc CPU utilization
	// The 5-second line reads: "CPU utilization for five seconds: 4%/0%; ..."
	// The previous greedy regex captured the *last* percentage on the line.
	ciscoiosCPURe = regexp.MustCompile(`CPU utilization for five seconds:\s*(\d+)%`)

	// show processes memory | include Processor Pool
	// Output: "Processor Pool Total:  131073608 Used:  105684720 Free:   25388888"
	ciscoiosRAMRe = regexp.MustCompile(`Total:\s*(\d+)\s+Used:\s*\d+\s+Free:\s*(\d+)`)
)

type CiscoIOSDriver struct {
	*driver.Driver
}

// showVersion runs "show version" and returns the output. Several providers
// (GetVersion, GetVersionBootROM, GetSerialNumber, GetUptime) share this command.
func (d *CiscoIOSDriver) showVersion(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "show version")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

func (d *CiscoIOSDriver) GetHostname(ctx context.Context, timeout time.Duration) (string, error) {
	// "show running-config | include ^hostname" is unambiguous and fast.
	result, err := d.Cmd(ctx, timeout, "show running-config | include ^hostname")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	m := ciscoiosHostnameRe.FindStringSubmatch(result.Output)
	if len(m) < 2 {
		return "", fmt.Errorf("hostname not found in running config")
	}
	return m[1], nil
}

func (d *CiscoIOSDriver) GetVersion(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	out, err := d.showVersion(ctx, timeout)
	if err != nil {
		return nil, err
	}
	matches := ciscoiosVersionRe.FindAllStringSubmatch(out, -1)
	if len(matches) == 0 {
		return []*driver.FirmwareInfo{}, nil
	}
	infos := make([]*driver.FirmwareInfo, len(matches))
	for i, m := range matches {
		infos[i] = &driver.FirmwareInfo{Version: m[1]}
	}
	return infos, nil
}

func (d *CiscoIOSDriver) GetVersionBootROM(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	out, err := d.showVersion(ctx, timeout)
	if err != nil {
		return nil, err
	}
	matches := ciscoiosBootromRe.FindAllStringSubmatch(out, -1)
	if len(matches) == 0 {
		return []*driver.FirmwareInfo{}, nil
	}
	infos := make([]*driver.FirmwareInfo, len(matches))
	for i, m := range matches {
		infos[i] = &driver.FirmwareInfo{Version: m[1]}
	}
	return infos, nil
}

// GetSerialNumber returns one serial per stack member. Uses FindAll so a
// stacked switch with repeated "System serial number" lines returns all of them.
// Falls back to "Processor board ID" on older IOS that lacks the labeled field.
func (d *CiscoIOSDriver) GetSerialNumber(ctx context.Context, timeout time.Duration) ([]string, error) {
	out, err := d.showVersion(ctx, timeout)
	if err != nil {
		return nil, err
	}
	matches := ciscoiosSerialRe.FindAllStringSubmatch(out, -1)
	if len(matches) == 0 {
		// Older IOS (e.g. 12.x) uses "Processor board ID <serial>"
		matches = ciscoiosSerialOldRe.FindAllStringSubmatch(out, -1)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("serial number not found in show version")
	}
	serials := make([]string, len(matches))
	for i, m := range matches {
		serials[i] = m[1]
	}
	return serials, nil
}

func (d *CiscoIOSDriver) GetCPU(ctx context.Context, timeout time.Duration) (int, error) {
	result, err := d.Cmd(ctx, timeout, "show processes cpu | inc CPU utilization")
	if err != nil {
		return 0, err
	}
	if result.Failed {
		return 0, fmt.Errorf("device error: %s", result.FailMsg)
	}
	m := ciscoiosCPURe.FindStringSubmatch(result.Output)
	if len(m) < 2 {
		return 0, fmt.Errorf("CPU utilization not found in output")
	}
	return strconv.Atoi(m[1])
}

func (d *CiscoIOSDriver) GetRAM(ctx context.Context, timeout time.Duration) (*driver.RAMInfo, error) {
	result, err := d.Cmd(ctx, timeout, "show processes memory | include Processor Pool")
	if err != nil {
		return nil, err
	}
	if result.Failed {
		return nil, fmt.Errorf("device error: %s", result.FailMsg)
	}
	m := ciscoiosRAMRe.FindStringSubmatch(result.Output)
	if len(m) < 3 {
		return nil, fmt.Errorf("memory info not found in show processes memory")
	}
	total, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse memory total: %w", err)
	}
	free, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse memory free: %w", err)
	}
	return &driver.RAMInfo{Total: total, Free: free}, nil
}

func (d *CiscoIOSDriver) GetUptime(ctx context.Context, timeout time.Duration) (int64, error) {
	out, err := d.showVersion(ctx, timeout)
	if err != nil {
		return 0, err
	}
	m := ciscoiosUptimeRe.FindStringSubmatch(out)
	if len(m) < 2 {
		return 0, fmt.Errorf("uptime not found in show version")
	}
	secs, err := driver.ParseUptimeSeconds(m[1])
	if err != nil {
		return 0, fmt.Errorf("parse uptime: %w", err)
	}
	return secs, nil
}

func (d *CiscoIOSDriver) GetEventLog(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "show logging")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

func (d *CiscoIOSDriver) GetConfig(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "show running-config")
	if err != nil {
		return "", err
	}
	if result.Failed {
		return "", fmt.Errorf("device error: %s", result.FailMsg)
	}
	return result.Output, nil
}

var ciscoIOSProfile = driver.Profile{
	PrivPrompt:     regexp.MustCompile(`\S+#\s*$`),
	UserPrompt:     regexp.MustCompile(`\S+>\s*$`),
	EnableCmd:      "enable",
	PasswordPrompt: regexp.MustCompile(`(?i)password:\s*$`),
	Errors: []*regexp.Regexp{
		regexp.MustCompile(`% Invalid input detected`),
		regexp.MustCompile(`% Ambiguous command`),
		regexp.MustCompile(`% Incomplete command`),
		regexp.MustCompile(`% Unknown command`),
	},
	DisablePaging: "terminal length 0",
}

func init() {
	driver.Register("ciscoios", func(s device.Settings) device.Device {
		return &CiscoIOSDriver{
			&driver.Driver{Settings: s, Profile: ciscoIOSProfile},
		}
	})
}
