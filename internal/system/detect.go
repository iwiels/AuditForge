package system

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func GetBinTargetDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".orquestador-auditor", "bin")
}



type ToolStatus struct {
	Name      string
	Installed bool
	Path      string
}

type PlatformProfile struct {
	OS             string
	Arch           string
	LinuxDistro    string
	PackageManager string
	Supported      bool
	HomeDir        string
}

type DetectionResult struct {
	Profile PlatformProfile
	Tools   map[string]ToolStatus
}

func Detect(ctx context.Context) (DetectionResult, error) {
	_ = ctx
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return DetectionResult{}, err
	}
	profile := PlatformProfile{
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		LinuxDistro:    detectLinuxDistro(runtime.GOOS),
		PackageManager: detectPackageManager(),
		Supported:      IsSupportedOS(runtime.GOOS),
		HomeDir:        homeDir,
	}
	tools := DetectTools([]string{"git", "npm", "brew", "winget"})
	return DetectionResult{Profile: profile, Tools: tools}, nil
}


func IsSupportedOS(goos string) bool {
	return goos == "darwin" || goos == "linux" || goos == "windows"
}

func DetectTools(names []string) map[string]ToolStatus {
	out := make(map[string]ToolStatus, len(names))
	binDir := GetBinTargetDir()
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			// Check local bin dir with multiple extensions on Windows
			extensions := []string{""}
			if runtime.GOOS == "windows" {
				extensions = []string{".exe", ".bat", ".ps1"}
			}
			for _, ext := range extensions {
				localPath := filepath.Join(binDir, name+ext)
				if _, statErr := os.Stat(localPath); statErr == nil {
					path = localPath
					err = nil
					break
				}
			}
		}
		out[name] = ToolStatus{Name: name, Installed: err == nil, Path: path}
	}
	return out
}

func detectPackageManager() string {
	switch {
	case hasBinary("brew"):
		return "brew"
	case hasBinary("apt-get"):
		return "apt"
	case hasBinary("pacman"):
		return "pacman"
	case hasBinary("dnf"):
		return "dnf"
	case hasBinary("winget"):
		return "winget"
	default:
		return ""
	}
}

func detectLinuxDistro(goos string) string {
	if goos != "linux" {
		return ""
	}
	raw, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "unknown"
	}
	content := strings.ToLower(string(raw))
	switch {
	case strings.Contains(content, "id=ubuntu"):
		return "ubuntu"
	case strings.Contains(content, "id=debian"):
		return "debian"
	case strings.Contains(content, "id=arch"):
		return "arch"
	case strings.Contains(content, "id=fedora"):
		return "fedora"
	default:
		return "unknown"
	}
}

func hasBinary(name string) bool {
	_, err := exec.LookPath(name)
	if err == nil {
		return true
	}
	binDir := GetBinTargetDir()
	extensions := []string{""}
	if runtime.GOOS == "windows" {
		extensions = []string{".exe", ".bat", ".ps1"}
	}
	for _, ext := range extensions {
		localPath := filepath.Join(binDir, name+ext)
		if _, err := os.Stat(localPath); err == nil {
			return true
		}
	}
	return false
}
