package tunnel

import (
	"os"
	"path/filepath"
)

type windowsPlatform struct{}

func init() {
	// Use Windows-style path
	BINARY_PATH = filepath.Join(os.Getenv("PROGRAMFILES"), "SST", "tunnel.exe")
	VERSION_PATH = filepath.Join(os.Getenv("PROGRAMFILES"), "SST", "tunnel.version")
	impl = &windowsPlatform{}
}

func (p *windowsPlatform) destroy() error {
	return nil
}

func (p *windowsPlatform) start(routes ...string) error {
	// Windows-specific installation
	return nil
}

func (p *windowsPlatform) isRunning() bool {
	return false
}

// Override Install for Windows
func (p *windowsPlatform) install() error {
	// Windows-specific installation
	// TODO: implement version file writing when Windows tunnel is implemented
	return nil
}
