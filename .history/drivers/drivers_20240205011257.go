package drivers

import (
	"fmt"

	"github.com/goodieshq/gnat/device"
)

type DeviceFactory func(device.DeviceSettings) device.Device
type DeviceSwitchFactory func(device.DeviceSettings) device.SwitchDevice

// var devices map[string]DeviceFactory
var deviceSwitches map[string]DeviceSwitchFactory

func RegisterDeviceSwitch(name string, factory DeviceSwitchFactory) error {
	if _, found := deviceSwitches[name]; found {
		return fmt.Errorf("driver '%s' is already registered", name)
	}

	deviceSwitches[name] = factory
	return nil
}

func DriverDeviceSwitch(name string) (DeviceSwitchFactory, error) {
	if f, found := deviceSwitches[name]; found {
		return f, nil
	}

}

func UnregisterDeviceSwitch(name string) error {
	delete(deviceSwitches, name)
	return nil
}
