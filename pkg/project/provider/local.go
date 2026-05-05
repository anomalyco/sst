package provider

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sst/sst/v3/internal/util"
	"github.com/sst/sst/v3/pkg/global"
)

type LocalHome struct {
}

func NewLocalHome() *LocalHome {
	return &LocalHome{}
}

func (l *LocalHome) Bootstrap() error {
	return nil
}

func (l *LocalHome) cleanup(key, app, stage string) error {
	return nil
}

func (l *LocalHome) getData(key, app, stage string) (io.Reader, error) {
	p := l.pathForData(key, app, stage)
	result, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return result, nil
}

func (l *LocalHome) putData(key, app, stage string, data io.Reader) error {
	if key == "summary" {
		return nil
	}
	p := l.pathForData(key, app, stage)
	err := os.MkdirAll(filepath.Dir(p), 0755)
	if err != nil {
		return err
	}
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, data)
	if err != nil {
		return err
	}
	return nil
}

func (l *LocalHome) removeData(key, app, stage string) error {
	p := l.pathForData(key, app, stage)
	return os.Remove(p)
}

func (l *LocalHome) purge(app, stage string) error {
	// Single-file keys.
	for _, key := range []string{"app", "secret"} {
		p := l.pathForData(key, app, stage)
		if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	// Folder-style keys.
	for _, key := range []string{"update", "summary", "eventlog", "snapshot"} {
		dir := filepath.Join(global.ConfigDir(), "state", key, app, stage)
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
	}
	return nil
}

func (c *LocalHome) removePassphrase(app, stage string) error {
	return c.removeData("passphrase", app, stage)
}

// these should go into secrets manager once it's out of beta
func (c *LocalHome) setPassphrase(app, stage string, passphrase string) error {
	return c.putData("passphrase", app, stage, bytes.NewReader([]byte(passphrase)))
}

func (c *LocalHome) getPassphrase(app, stage string) (string, error) {
	data, err := c.getData("passphrase", app, stage)
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", nil
	}
	read, err := io.ReadAll(data)
	if err != nil {
		return "", err
	}
	return string(read), nil
}

func (l *LocalHome) pathForData(key, app, stage string) string {
	return filepath.Join(global.ConfigDir(), "state", key, app, fmt.Sprintf("%v.json", stage))
}

func (a *LocalHome) listStages(app string) ([]string, error) {
	path := filepath.Join(global.ConfigDir(), "state", "app", app)

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var stages []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			stageName := strings.TrimSuffix(entry.Name(), ".json")
			if hasResources(a, app, stageName) {
				stages = append(stages, stageName)
			}
		}
	}

	return stages, nil
}

func (c *LocalHome) info() (util.KeyValuePairs[string], error) {
	return util.KeyValuePairs[string]{
		{Key: "Provider", Value: "Local"},
		{Key: "Path", Value: global.ConfigDir()},
	}, nil
}
