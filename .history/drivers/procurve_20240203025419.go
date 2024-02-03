package drivers

import (
	"bytes"
	"context"
	"regexp"

	"github.com/goodieshq/gnat/device"
)

type ProcurveDevice struct {
	device.DeviceSettings
}

// remote ansi escape sequences, from stripansi package
var ansi = regexp.MustCompile("[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))")

// all procurve versions seem to have 5 ansi escape sequences following the "#" from the prompt
var sequences = regexp.MustCompile(`#\s+(\x1b\[(\??)\d+(;?)\d+[a-zA-Z]){5}`)

func (procurve *ProcurveDevice) Sanitize(output []byte) string {
	var data []byte
	if len(output) == 0 {
		return ""
	}

	data = ansi.ReplaceAll(output, nil)
	return string(data)
}

func (procurve *ProcurveDevice) RegexInit() *regexp.Regexp {
	return sequences
}

func (procurve *ProcurveDevice) RegexCmd() *regexp.Regexp {
	return sequences
}

func (procurve *ProcurveDevice) Initialize(ctx context.Context) error {
	procurve.Connection.Send([]byte{'\n'})
	return procurve.DisablePaging(ctx)
}

func (procurve *ProcurveDevice) DisablePaging(ctx context.Context) error {
	result, err := procurve.Cmd(ctx, "no page")
	if err != nil {
		return err
	}

	return result.Err
}

func (procurve *ProcurveDevice) Cmd(ctx context.Context, command string) (*device.DeviceResult, error) {
	// write the command to the buffer along with a newline
	var buf bytes.Buffer
	buf.WriteString(command)
	buf.WriteByte('\n')

	procurve.Connection.Send(buf.Bytes())

	// read until the desired regex
	data, err := procurve.Connection.ReadUntilMatch(ctx, procurve.TimeoutRead)
	if err != nil {
		return nil, err
	}

	return &device.DeviceResult{
		Output:  procurve.Sanitize(data),
		Command: command,
	}, nil
}
