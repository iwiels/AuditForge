package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type Updater struct {
	Repo   string
	Client *http.Client
}

type Release struct {
	TagName string `json:"tag_name"`
}

func New(repo string) Updater {
	if strings.TrimSpace(repo) == "" {
		repo = "victo/orquestador_auditor"
	}
	return Updater{Repo: repo, Client: http.DefaultClient}
}

func (u Updater) LatestVersion() (string, error) {
	release := Release{}
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.Repo), nil)
	if err != nil {
		return "", err
	}
	resp, err := u.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("release lookup failed with status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	if strings.TrimSpace(release.TagName) == "" {
		return "", fmt.Errorf("latest release tag is empty")
	}
	return release.TagName, nil
}

func (u Updater) AssetName(version string) (string, error) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	version = strings.TrimPrefix(version, "v")
	switch goos {
	case "windows":
		return fmt.Sprintf("orquestador-auditor_%s_%s_%s.zip", version, goos, goarch), nil
	case "linux", "darwin":
		return fmt.Sprintf("orquestador-auditor_%s_%s_%s.tar.gz", version, goos, goarch), nil
	default:
		return "", fmt.Errorf("unsupported platform %s/%s", goos, goarch)
	}
}

func (u Updater) Download(version string) ([]byte, error) {
	asset, err := u.AssetName(version)
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", u.Repo, version, asset)
	resp, err := u.Client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (u Updater) Apply(version string) (string, error) {
	raw, err := u.Download(version)
	if err != nil {
		return "", err
	}
	binary, err := extractBinary(raw)
	if err != nil {
		return "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	target := exe
	if runtime.GOOS == "windows" {
		target = exe + ".new"
	}
	if err := os.WriteFile(target, binary, 0o755); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(target, 0o755); err != nil {
			return "", err
		}
	}
	return target, nil
}

func extractBinary(archive []byte) ([]byte, error) {
	if runtime.GOOS == "windows" {
		reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
		if err != nil {
			return nil, err
		}
		for _, file := range reader.File {
			if strings.EqualFold(filepath.Base(file.Name), "orquestador-auditor.exe") {
				rc, err := file.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf("binary not found in zip archive")
	}
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if filepath.Base(header.Name) == "orquestador-auditor" {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("binary not found in archive")
}
