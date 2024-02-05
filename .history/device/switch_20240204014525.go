package device

import "context"

type SwitchDevice interface {
	GetRunningConfig(context.Context) (string, error)
}
