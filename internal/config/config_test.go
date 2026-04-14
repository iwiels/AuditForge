package config

import "testing"

func TestParseUsesPositionalTarget(t *testing.T) {
	cfg, err := Parse([]string{"./repo"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Target != "./repo" {
		t.Fatalf("target = %q", cfg.Target)
	}
}

func TestParseUsesTargetFlag(t *testing.T) {
	cfg, err := Parse([]string{"--target", "./repo", "--provider", "static"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Target != "./repo" {
		t.Fatalf("target = %q", cfg.Target)
	}
}

func TestParseErrorsWithoutTarget(t *testing.T) {
	if _, err := Parse(nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseAdvancedProfileFlags(t *testing.T) {
	cfg, err := Parse([]string{"--target", "https://example.com", "--profile", "full", "--enable-arjun", "--enable-waymore"})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cfg.Profile != "full" || !cfg.EnableArjun || !cfg.EnableWaymore {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestResolveExecutionProfile(t *testing.T) {
	profile := ResolveExecutionProfile(Config{Profile: "archive-forensics"})
	if profile.EnableBrowserCapture || profile.EnableArjun || !profile.EnableWaymore {
		t.Fatalf("unexpected execution profile: %#v", profile)
	}
}
