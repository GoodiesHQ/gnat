package drivers

import (
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
	var sdata []byte
	if len(output) == 0 {
		return ""
	}

	sdata = ansi.ReplaceAll(output, nil)
	return string(sdata)
}
