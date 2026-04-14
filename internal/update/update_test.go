package update

import (
	"runtime"
	"strings"
	"testing"
)

func TestAssetNameMatchesPlatformArchive(t *testing.T) {
	name, err := New("victo/orquestador_auditor").AssetName("v1.2.3")
	if err != nil {
		t.Fatalf("AssetName() error = %v", err)
	}
	if runtime.GOOS == "windows" {
		if !strings.HasSuffix(name, ".zip") {
			t.Fatalf("expected zip asset, got %s", name)
		}
		return
	}
	if !strings.HasSuffix(name, ".tar.gz") {
		t.Fatalf("expected tar.gz asset, got %s", name)
	}
}
