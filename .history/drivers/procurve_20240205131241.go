package drivers

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/goodieshq/gnat/device"
	"github.com/goodieshq/gnat/utils"
	"github.com/rs/zerolog/log"
)

type ProcurveDevice struct {
	device.DeviceSettings
	memberMIBs []string // MIB IDs for all members of a stack
}

func NewProcurveDevice(settings device.DeviceSettings) device.DeviceSwitch {
	return &ProcurveDevice{DeviceSettings: settings}
}

func RegisterProcurve() error {
	return RegisterDeviceSwitch("procurve", NewProcurveDevice)
}

// remote ansi escape sequences, from stripansi package
var ansi = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// all procurve versions seem to have 5 ansi escape sequences following the "#" from the prompt
var sequences = regexp.MustCompile(`#\s+(\x1b\[(\??)\d+(;?)\d+[a-zA-Z]){5}`)

func (procurve *ProcurveDevice) GetValueMIB(ctx context.Context, mib string) (string, error) {
	_, v, err := procurve.GetMIB(ctx, mib)
	return v, err
}

func (procurve *ProcurveDevice) GetMembers(ctx context.Context) ([]string, error) {
	if procurve.memberMIBs != nil {
		return procurve.memberMIBs, nil
	}

	keys, vals, err := procurve.WalkMIB(ctx, "entPhysicalName")
	if err != nil {
		return nil, err
	}

	for i, val := range vals {
		// find only physical devices with the type "chassis"
		if strings.ToLower(val) == "chassis" {
			if strings.Count(keys[i], ".") != 1 {
				return nil, fmt.Errorf("invalid MIB name returned while walking")
			}
			id := strings.Split(keys[i], ".")[1]
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
	}

}

func (procurve *ProcurveDevice) GetMIB(ctx context.Context, mib string) (string, string, error) {
	result, err := procurve.Cmd(ctx, procurve.TimeoutRead, fmt.Sprintf("getMIB %s", mib))
	if err != nil {
		return "", "", err
	}

	sep := " ="

	if strings.Count(result.Output, sep) != 1 {
		return "", "", fmt.Errorf("invalid MIB '%s'", result.Output)
	}

	vals := strings.Split(result.Output, sep)
	return strings.TrimSpace(vals[0]), strings.TrimSpace(vals[1]), nil
}

func (procurve *ProcurveDevice) WalkMIB(ctx context.Context, mib string) ([]string, []string, error) {
	result, err := procurve.Cmd(ctx, procurve.TimeoutRead, fmt.Sprintf("walkMIB %s", mib))
	if err != nil {
		return nil, nil, err
	}

	sep := " ="
	lines := utils.SplitLines(result.Output)

	keys := make([]string, len(lines))
	vals := make([]string, len(lines))

	for _, line := range lines {
		if strings.Count(line, sep) != 1 {
			return nil, nil, fmt.Errorf("invalid output '%s'", line)
		}
		kv := strings.Split(line, sep)

		k, v := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])

		keys = append(keys, k)
		vals = append(vals, v)
	}
	return keys, vals, nil
}

func (procurve *ProcurveDevice) Sanitize(output []byte) string {
	var data []byte
	if len(output) == 0 {
		return ""
	}

	data = ansi.ReplaceAll(output, nil)

	idx := bytes.LastIndex(data, []byte("\n"))
	if idx < 0 {
		return ""
	}
	data = data[:idx+1]
	return string(data)
}

func (procurve *ProcurveDevice) RegexInit() *regexp.Regexp {
	return sequences
}

func (procurve *ProcurveDevice) RegexCmd() *regexp.Regexp {
	return sequences
}

func (procurve *ProcurveDevice) Initialize(ctx context.Context) error {
	if err := procurve.Connection.Send([]byte{'\n'}); err != nil {
		return err
	}

	if data, err := procurve.Connection.ReadUntilMatch(ctx, procurve.TimeoutRead, sequences); err != nil {
		return err
	} else {
		log.Info().Bytes("data", data).Send()
	}

	if err := procurve.DisablePaging(ctx); err != nil {
		return err
	}

	return nil
}

func (procurve *ProcurveDevice) FlushFor(ctx context.Context, t time.Duration) error {
	return procurve.Connection.FlushFor(ctx, t)
}

func (procurve *ProcurveDevice) DisablePaging(ctx context.Context) error {
	x, err := procurve.Cmd(ctx, procurve.TimeoutRead, "no page")
	if err != nil {
		log.Info().Str("output", x.Output).Msg("failed to disable paging")
		return err
	}
	log.Info().Str("output", x.Output).Msg("disabled paging")
	return err
}

func (procurve *ProcurveDevice) Cmd(ctx context.Context, timeout time.Duration, command string) (*device.DeviceResult, error) {
	// write the command to the buffer along with a newline
	var buf bytes.Buffer
	buf.WriteString(command)
	buf.WriteByte('\n')

	procurve.Connection.Send(buf.Bytes())

	// read until the desired regex
	data, err := procurve.Connection.ReadUntilMatch(ctx, timeout, sequences)
	if err != nil {
		return nil, err
	}

	sdata := strings.TrimPrefix(procurve.Sanitize(data), command)

	return &device.DeviceResult{
		Output:  sdata,
		Command: command,
	}, nil
}

func (procurve *ProcurveDevice) GetRunningConfig(ctx context.Context) (string, error) {
	result, err := procurve.Cmd(ctx, procurve.TimeoutRead, "write terminal")
	if err != nil {
		return "", err
	}
	return utils.JoinLines(utils.SplitLines(result.Output)), nil
}

func (procurve *ProcurveDevice) GetLogs(ctx context.Context) (string, error) {
	result, err := procurve.Cmd(ctx, procurve.TimeoutRead, "show log -r")
	if err != nil {
		return "", err
	}
	return utils.JoinLines(utils.SplitLines(result.Output)), nil
}

func (procurve *ProcurveDevice) GetVersion(ctx context.Context) (string, error) {
	return procurve.GetValueMIB(ctx, "hpHttpMgVersion.0")
}

func (procurve *ProcurveDevice) GetBootROMVersion(ctx context.Context) (string, error) {
	return procurve.GetValueMIB(ctx, "hpHttpMgROMVersion.0")
}

func (procurve *ProcurveDevice) GetCPU(ctx context.Context) (int, error) {
	s, err := procurve.GetValueMIB(ctx, "hpSwitchCpuStat.0")
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(strings.Replace(s, ",", "", -1))
}

func (procurve *ProcurveDevice) GetRAM(ctx context.Context) (int, error) {
	memAllocStr, err := procurve.GetValueMIB(ctx, "hpLocalMemAllocBytes.1")
	if err != nil {
		return -1, err
	}

	memTotalStr, err := procurve.GetValueMIB(ctx, "hpLocalMemTotalBytes.1")
	if err != nil {
		return -1, err
	}

	memAlloc, err := strconv.Atoi(strings.ReplaceAll(memAllocStr, ",", ""))
	if err != nil {
		return -1, err
	}

	memTotal, err := strconv.Atoi(strings.ReplaceAll(memTotalStr, ",", ""))
	if err != nil {
		return -1, err
	}

	return (100 * memAlloc / memTotal), nil
}

func (procurve *ProcurveDevice) GetUptime(ctx context.Context) (string, error) {
	return procurve.GetValueMIB(ctx, "sysUpTime.0")
}

func (procurve *ProcurveDevice) GetSysname(ctx context.Context) (string, error) {
	return procurve.GetValueMIB(ctx, "sysName.0")
}

func (procurve *ProcurveDevice) GetSerialNumbers(ctx context.Context) ([]string, error) {
	ids := make([]string, 0)
	sns := make([]string, 0)

	for _, id := range ids {
		sn, err := procurve.GetValueMIB(ctx, "entPhysicalSerialNum."+id)
		if err != nil {
			return nil, err
		}
		sns = append(sns, sn)
	}

	return sns, nil
}
