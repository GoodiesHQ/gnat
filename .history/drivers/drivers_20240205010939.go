package drivers

import "github.com/goodieshq/gnat/device"

type deviceFactory func(device.DeviceSettings) device.Device
type deviceSwitchFactory func(device.DeviceSettings) device.SwitchDevice

var devices map[string]deviceFactory
var deviceSwitches map[string]deviceSwitchFactory
