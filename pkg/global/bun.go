package global

import (
	"archive/zip"
	"bytes"
	"context"
	"debug/elf"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/klauspost/cpuid/v2"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/id"
	"github.com/sst/sst/v3/pkg/process"
	"github.com/sst/sst/v3/pkg/task"
)

func BunPath() string {
	path := filepath.Join(BinPath(), "bun")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	return path
}

func NeedsBun() bool {
	if flag.SST_NO_BUN {
		return false
	}
	path := BunPath()
	slog.Info("checking for bun", "path", path)
	if _, err := os.Stat(path); err != nil {
		return true
	}
	cmd := process.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return true
	}
	version := strings.TrimSpace(string(output))
	return version != BUN_VERSION
}

func InstallBun(ctx context.Context) error {
	slog.Info("bun install")
	bunPath := BunPath()

	goos := runtime.GOOS
	arch := runtime.GOARCH

	isMusl := false
	if goos == "linux" {
		isMusl = linuxUsesMusl()
	}

	var filename string
	switch {
	case goos == "darwin" && arch == "arm64":
		filename = "bun-darwin-aarch64.zip"
	case goos == "darwin" && arch == "amd64":
		filename = "bun-darwin-x64-baseline.zip"
		if cpuid.CPU.Has(cpuid.AVX2) {
			filename = "bun-darwin-x64.zip"
		}
	case goos == "linux" && arch == "arm64":
		filename = "bun-linux-aarch64.zip"
		if isMusl {
			filename = "bun-linux-aarch64-musl.zip"
		}
	case goos == "linux" && arch == "amd64":
		filename = "bun-linux-x64-baseline.zip"
		if cpuid.CPU.Has(cpuid.AVX2) {
			filename = "bun-linux-x64.zip"
		}
		if isMusl {
			if cpuid.CPU.Has(cpuid.AVX2) {
				filename = "bun-linux-x64-musl.zip"
			}
			filename = "bun-linux-x64-musl-baseline.zip"
		}
	case goos == "windows" && arch == "amd64":
		filename = "bun-windows-x64-baseline.zip"
		if cpuid.CPU.Has(cpuid.AVX2) {
			filename = "bun-windows-x64.zip"
		}
	default:
	}
	if filename == "" {
		return fmt.Errorf("unsupported platform: %s %s", goos, arch)
	}
	slog.Info("bun selected", "filename", filename)

	_, err := task.Run(ctx, func() (any, error) {
		url := "https://github.com/oven-sh/bun/releases/download/bun-v" + BUN_VERSION + "/" + filename
		slog.Info("bun downloading", "url", url)
		response, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("bad status: %s", response.Status)
		}
		bodyBytes, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}
		readerAt := bytes.NewReader(bodyBytes)
		zipReader, err := zip.NewReader(readerAt, readerAt.Size())
		if err != nil {
			return nil, err
		}
		for _, file := range zipReader.File {
			filename := filepath.Base(file.Name)
			if filename == "bun" || filename == "bun.exe" {
				f, err := file.Open()
				if err != nil {
					return nil, err
				}
				defer f.Close()

				tmpFile := filepath.Join(BinPath(), id.Ascending())
				outFile, err := os.Create(tmpFile)
				if err != nil {
					return nil, err
				}
				defer outFile.Close()

				_, err = io.Copy(outFile, f)
				if err != nil {
					return nil, err
				}
				err = outFile.Close()
				if err != nil {
					return nil, err
				}

				err = os.Rename(tmpFile, bunPath)
				if err != nil {
					return nil, err
				}

				err = os.Chmod(bunPath, 0755)
				if err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	})
	return err
}

func linuxUsesMusl() bool {
	isMusl, ok := muslFromLdd()
	if ok {
		return isMusl
	}

	for _, path := range []string{"/bin/sh", "/usr/bin/env", "/bin/busybox"} {
		isMusl, ok := muslFromELF(path)
		if ok {
			return isMusl
		}
	}

	isMusl, ok = muslFromAlpineRelease()
	if ok {
		return isMusl
	}

	isMusl, ok = muslFromLoaderPath()
	if ok {
		return isMusl
	}

	return false
}

func muslFromLdd() (bool, bool) {
	lddPath, err := exec.LookPath("ldd")
	if err != nil {
		return false, false
	}

	output, err := exec.Command(lddPath, "--version").CombinedOutput()
	if err != nil && len(output) == 0 {
		return false, false
	}

	return strings.Contains(strings.ToLower(string(output)), "musl"), true
}

func muslFromELF(path string) (bool, bool) {
	interpreter, err := elfInterpreter(path)
	if err != nil || interpreter == "" {
		return false, false
	}

	return strings.Contains(strings.ToLower(interpreter), "musl"), true
}

func muslFromAlpineRelease() (bool, bool) {
	_, err := os.Stat("/etc/alpine-release")
	if err == nil {
		return true, true
	}

	return false, false
}

func muslFromLoaderPath() (bool, bool) {
	var path string
	switch runtime.GOARCH {
	case "amd64":
		path = "/lib/ld-musl-x86_64.so.1"
	case "arm64":
		path = "/lib/ld-musl-aarch64.so.1"
	default:
		return false, false
	}

	_, err := os.Stat(path)
	if err == nil {
		return true, true
	}

	return false, false
}

func elfInterpreter(path string) (string, error) {
	file, err := elf.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	for _, prog := range file.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		interpreter, err := io.ReadAll(prog.Open())
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(interpreter), "\x00"), nil
	}

	return "", fmt.Errorf("no ELF interpreter found for %s", path)
}
