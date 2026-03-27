package tunnel

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/xjasonlyu/tun2socks/v2/engine"

	"github.com/sst/sst/v3/pkg/process"
)

// Version of the tunnel binary. Bump this when tunnel code changes and needs to be re-installed.
const Version = "2"

func IsRunning() bool {
	return impl.isRunning()
}

// Platform-specific interface
type tunnelPlatform interface {
	destroy() error
	start(routes ...string) error
	install() error
	isRunning() bool
}

var impl tunnelPlatform

var BINARY_PATH = "/opt/sst/tunnel"
var VERSION_PATH = "/opt/sst/tunnel.version"

func NeedsInstall() bool {
	if _, err := os.Stat(BINARY_PATH); err != nil {
		return true
	}

	checkVersion := Version
	if testVersion := os.Getenv("SST_TEST_TUNNEL_VERSION"); testVersion != "" {
		checkVersion = testVersion
	}

	versionBytes, err := os.ReadFile(VERSION_PATH)
	if err != nil {
		return true
	}

	installedVersion := strings.TrimSpace(string(versionBytes))
	return installedVersion != checkVersion
}

func Install() error {
	return impl.install()
}

func runCommands(cmds [][]string) error {
	for _, item := range cmds {
		slog.Info("running command", "command", item)
		cmd := process.Command(item[0], item[1:]...)
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to execute command", "command", item, "error", err)
			return fmt.Errorf("failed to execute command '%v': %v", item, err)
		}
	}
	return nil
}

func Start(routes ...string) error {
	return impl.start(routes...)
}

func tun2socks(name string) {
	key := new(engine.Key)
	key.Device = name
	key.Proxy = "socks5://127.0.0.1:1080"
	engine.Insert(key)
	engine.Start()
}

func Stop() {
	engine.Stop()
	impl.destroy()
}
