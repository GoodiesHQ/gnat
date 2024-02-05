package device

import "context"

type SwitchDevice interface {
	GetRunningConfig(context.Context) (string, error)  // get the current configuration
	GetVersion(context.Context) (string, error)        // get the current running software version
	GetBootROMVersion(context.Context) (string, error) // get the current ROM version
	GetCPU(context.Context) (map[int]float32, error)   //
}
