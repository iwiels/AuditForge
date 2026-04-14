package system

import (
	"fmt"
	"runtime"
)

func EnsureCurrentOSSupported() error {
	if !IsSupportedOS(runtime.GOOS) {
		return fmt.Errorf("unsupported operating system %q", runtime.GOOS)
	}
	return nil
}
