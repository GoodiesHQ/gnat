package drivers

import (
	"fmt"

	"github.com/goodieshq/gnat/device"
)

type DeviceFactory func(device.DeviceSettings) device.Device
type DeviceSwitchFactory func(device.DeviceSettings) device.SwitchDevice

// var deviceRegistry map[string]DeviceFactory
var deviceSwitchRegistry map[string]DeviceSwitchFactory

func RegisterDeviceSwitch(name string, factory DeviceSwitchFactory) error {
	if _, found := deviceSwitchRegistry[name]; found {
		return fmt.Errorf("driver '%s' is already registered", name)
	}

	deviceSwitchRegistry[name] = factory
	return nil
}

func DriverDeviceSwitch(name string) (DeviceSwitchFactory, error) {
	if f, found := deviceSwitchRegistry[name]; found {
		return f, nil
	}
	return nil, fmt.Errorf("driver '%s' cannot be found", name)
}

func UnregisterDeviceSwitch(name string) error {
	delete(deviceSwitchRegistry, name)
	return nil
}
