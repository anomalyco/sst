package project

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/sst/sst/v3/platform"
)

func (p *Project) CheckPlatform(version string) bool {
	if version == "dev" {
		currentExecutable, _ := os.Executable()
		info, _ := os.Stat(currentExecutable)
		version = fmt.Sprint(info.ModTime().UnixMilli())
	}
	slog.Info("checking platform")
	contents, err := os.ReadFile(filepath.Join(p.PathPlatformSST(), "version"))
	if err != nil {
		slog.Error(err.Error())
		return false
	}
	return string(contents) == version
}

func (p *Project) CopyPlatform(version string) error {
	slog.Info("installing platform")
	os.RemoveAll(p.PathPlatformDir())
	platformDir := p.PathPlatformSST()
	os.MkdirAll(platformDir, 0755)
	err := platform.CopyTo(".", platformDir)
	if err != nil {
		return err
	}
	if version == "dev" {
		currentExecutable, _ := os.Executable()
		info, _ := os.Stat(currentExecutable)
		version = fmt.Sprint(info.ModTime().UnixMilli())
	}
	return os.WriteFile(filepath.Join(platformDir, "version"), []byte(version), 0644)
}
