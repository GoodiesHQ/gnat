package drivers

import "github.com/goodieshq/gnat/device"

type DeviceFactory func(device.DeviceSettings) device.Device
type DeviceSwitchFactory func(device.DeviceSettings) device.SwitchDevice

var devices map[string]DeviceFactory
var deviceSwitches map[string]DeviceSwitchFactory

func RegisterDeviceSwitch(name string, factory DeviceSwitchFactory)
