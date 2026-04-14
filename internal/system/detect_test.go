package system

import "testing"

func TestIsSupportedOS(t *testing.T) {
	if !IsSupportedOS("linux") || !IsSupportedOS("darwin") || !IsSupportedOS("windows") {
		t.Fatalf("expected linux, darwin, and windows to be supported")
	}
	if IsSupportedOS("plan9") {
		t.Fatalf("unexpected support for plan9")
	}
}

func TestRequiredSecurityToolsIsNotEmpty(t *testing.T) {
	if len(RequiredSecurityTools()) == 0 {
		t.Fatalf("expected required security tools")
	}
}
