//go:build !linux
// +build !linux

package utils

import (
	"fmt"
)

// HandleServiceCommand is not supported on non-Linux platforms
func HandleServiceCommand(command, execPath, configPath string, logger *Logger) error {
	return fmt.Errorf("service management is only supported on Linux. Current platform does not support systemd")
}
