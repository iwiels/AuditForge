package installcmd

import (
	"testing"

	"orquestador-auditor/internal/system"
)

func TestResolveBundleInstallFull(t *testing.T) {
	r := runtimeResolver{profile: system.PlatformProfile{OS: "linux", PackageManager: "apt"}}
	commands, err := r.ResolveBundleInstall("full")
	if err != nil {
		t.Fatalf("ResolveBundleInstall() error = %v", err)
	}
	if len(commands) == 0 {
		t.Fatalf("expected bundle commands")
	}
}
