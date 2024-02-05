package drivers

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/goodieshq/gnat/device"
	"github.com/rs/zerolog/log"
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

	idx := strings.LastIndex(data, "\n")
	if idx < 0 {
		return ""
	}
	data = data[:idx+1]
}
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
		Output:  upToLastLine(sdata),
		Command: command,
	}, nil
}
