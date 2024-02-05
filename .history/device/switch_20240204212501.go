package device

import "context"

type SwitchDevice interface {
	GetRunningConfig(context.Context) (string, error) // get the current configuration
	GetVersion(context.Context) (string, error)       // get the current running software version
	GetROMVersion(context.Context) (string, error)    // get the current ROM versoin
}
