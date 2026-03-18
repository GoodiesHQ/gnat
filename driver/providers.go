package driver

import (
	"context"
	"time"
)

type FirmwareInfo struct {
	Version string
}

type RAMInfo struct {
	Total int64
	Free  int64
}

type ProviderFirmware interface {
	GetVersion(ctx context.Context, timeout time.Duration) ([]*FirmwareInfo, error)
	GetVersionBootROM(ctx context.Context, timeout time.Duration) ([]*FirmwareInfo, error)
}

type ProviderCPU interface {
	GetCPU(ctx context.Context, timeout time.Duration) (int, error)
}

type ProviderRAM interface {
	GetRAM(ctx context.Context, timeout time.Duration) (*RAMInfo, error)
}

type ProviderSerialNumber interface {
	GetSerialNumber(ctx context.Context, timeout time.Duration) (string, error)
}

type ProviderHostname interface {
	GetHostname(ctx context.Context, timeout time.Duration) (string, error)
}

type ProviderConfig interface {
	GetConfig(ctx context.Context, timeout time.Duration) (string, error)
}
