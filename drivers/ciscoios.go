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

var ciscoiosCPURe = regexp.MustCompile(`(?m)^\s*CPU.*(\d+)%`)
var ciscoiosVersionBootromRe = regexp.MustCompile(`(?m)^\s*BOOTLDR.*Version (\S+),`)
var ciscoiosVersionRe = regexp.MustCompile(`(?m)^\s*Cisco IOS Software.*Version (\S+),`)

type CiscoIOSDriver struct {
	*driver.Driver
}

func (d *CiscoIOSDriver) GetCPU(ctx context.Context, timeout time.Duration) (int, error) {
	result, err := d.Cmd(ctx, timeout, "show processes cpu | inc CPU")
	if err != nil {
		return 0, err
	}
	if result.Failed {
		return 0, fmt.Errorf("device error: %s", result.FailMsg)
	}
	matches := ciscoiosCPURe.FindAllStringSubmatch(result.Output, -1)
	if len(matches) < 1 || len(matches[0]) < 2 {
		return 0, fmt.Errorf("no CPU usage found in output")
	}
	return strconv.Atoi(matches[0][1])
}

func (d *CiscoIOSDriver) GetVersion(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	result, err := d.Cmd(ctx, timeout, "show version")
	if err != nil {
		return nil, err
	}
	if result.Failed {
		return nil, fmt.Errorf("device error: %s", result.FailMsg)
	}
	matches := ciscoiosVersionRe.FindAllStringSubmatch(result.Output, -1)
	if len(matches) == 0 {
		return []*driver.FirmwareInfo{}, nil
		// return nil, fmt.Errorf("no version found in output")
	}
	infos := make([]*driver.FirmwareInfo, len(matches))
	for i, m := range matches {
		infos[i] = &driver.FirmwareInfo{Version: m[1]}
	}
	return infos, nil
}

func (d *CiscoIOSDriver) GetVersionBootROM(ctx context.Context, timeout time.Duration) ([]*driver.FirmwareInfo, error) {
	result, err := d.Cmd(ctx, timeout, "show version")
	if err != nil {
		return nil, err
	}
	if result.Failed {
		return nil, fmt.Errorf("device error: %s", result.FailMsg)
	}
	matches := ciscoiosVersionBootromRe.FindAllStringSubmatch(result.Output, -1)
	if len(matches) == 0 {
		return []*driver.FirmwareInfo{}, nil
		// return nil, fmt.Errorf("no version found in output")
	}
	infos := make([]*driver.FirmwareInfo, len(matches))
	for i, m := range matches {
		infos[i] = &driver.FirmwareInfo{Version: m[1]}
	}
	return infos, nil
}

func (d *CiscoIOSDriver) GetConfig(ctx context.Context, timeout time.Duration) (string, error) {
	result, err := d.Cmd(ctx, timeout, "write terminal")
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
