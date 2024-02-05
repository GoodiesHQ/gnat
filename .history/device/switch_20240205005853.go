package device

import "context"

// Typical operations that all switches should be able to support
type SwitchDevice interface {
	// Initialize(context.Context) error                  // initialize the connection if needed
	Device
	GetRunningConfig(context.Context) (string, error)  // get the current configuration
	GetLogs(context.Context) (string, error)           // get the current configuration
	GetVersion(context.Context) (string, error)        // get the current running software version
	GetBootROMVersion(context.Context) (string, error) // get the current ROM version (if different from GetVersion)
	GetCPU(context.Context) (int, error)               // get CPU utilization percentage
	GetRAM(context.Context) (int, error)               // get RAM utilization percentage
	GetUptime(context.Context) (string, error)         // get the current uptime
	GetSysname(context.Context) (string, error)        // get the system's name
}
