package project

import (
	"io"
	"os"
	"path/filepath"

	"github.com/sst/sst/v3/pkg/project/path"
	"github.com/sst/sst/v3/pkg/project/provider"
)

func (p *Project) DownloadResource(hash string) (string, error) {
	name := hash + ".mjs"
	dir := path.ResolveResourceDir(p.PathConfig())
	destination := filepath.Join(dir, name)

	if _, err := os.Stat(destination); err == nil {
		return destination, nil
	}

	file, err := os.Create(destination)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader, err := provider.GetResource(p.home, hash)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(file, reader)
	if err != nil {
		return "", err
	}

	return destination, nil
}
