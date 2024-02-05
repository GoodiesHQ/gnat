package device

type SwitchDevice interface {
	GetRunningConfig() (string, error)
}
