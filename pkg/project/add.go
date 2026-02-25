package project

import (
	"bytes"
	"fmt"
	"github.com/sst/sst/v3/pkg/process"
	"os"
	"path/filepath"
	"strings"
)

func (p *Project) Add(pkg string, version string) error {
	var stderr bytes.Buffer
	cmd := process.Command("node", filepath.Join(p.PathPlatformDir(), "src/ast/add.mjs"),
		p.PathConfig(),
		pkg,
		version,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return err
	}
	return nil
}
