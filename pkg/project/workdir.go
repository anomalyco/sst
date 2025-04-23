package project

import (
	"encoding/json"
	"fmt"

	"os"
	"path/filepath"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/pkg/flag"
	"github.com/sst/sst/v3/pkg/project/provider"
	"github.com/zeebo/xxh3"
	"golang.org/x/sync/errgroup"
)

type PulumiWorkdir struct {
	path       string
	project    *Project
	lastPushed uint64
}

func (p *Project) NewWorkdir(id string) (*PulumiWorkdir, error) {
	workdir := PulumiWorkdir{
		path:    filepath.Join(p.PathWorkingDir(), "pulumi", id),
		project: p,
	}
	err := os.MkdirAll(workdir.path, 0755)
	if err != nil {
		return nil, err
	}
	return &workdir, nil
}

func (w *PulumiWorkdir) Cleanup() {
	if flag.SST_NO_CLEANUP {
		return
	}
	os.RemoveAll(w.path)
}

func (w *PulumiWorkdir) PushPartial(updateID string) error {
	statePath := w.state()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return err
	}
	return w.pushPartial(updateID, data)
}

func (w *PulumiWorkdir) pushPartial(updateID string, data []byte) error {
	home := w.project.Backend()
	app := w.project.app.Name
	stage := w.project.app.Stage
	next := xxh3.Hash(data)
	if next != uint64(w.lastPushed) && next != 0 {
		err := provider.PushPartialState(home, updateID, app, stage, data)
		if err != nil {
			return err
		}
	}
	w.lastPushed = next
	return nil
}

func (w *PulumiWorkdir) Push(updateID string) error {
	statePath := w.state()
	data, err := os.ReadFile(statePath)
	if err != nil {
		return err
	}
	stage := w.project.app.Stage
	app := w.project.app.Name
	home := w.project.Backend()

	var group errgroup.Group
	group.Go(func() error {
		return w.pushPartial(updateID, data)
	})
	group.Go(func() error {
		return provider.PushSnapshot(home, updateID, app, stage, data)
	})
	group.Go(func() error {
		file, err := os.Open(w.EventLogPath())
		if err != nil {
			return nil
		}
		return provider.PushEventLog(home, updateID, app, stage, file)
	})
	return group.Wait()
}

func (w *PulumiWorkdir) Pull() (string, error) {
	appDir := filepath.Join(w.path, ".pulumi", "stacks", w.project.app.Name)
	path := filepath.Join(appDir, fmt.Sprintf("%v.json", w.project.app.Stage))

	err := os.MkdirAll(appDir, 0755)
	if err != nil {
		return path, err
	}
	err = provider.PullState(
		w.project.home,
		w.project.app.Name,
		w.project.app.Stage,
		path,
	)
	if err != nil {
		return path, err
	}
	return path, nil
}

func (w *PulumiWorkdir) Backend() string {
	return w.path
}

func (w *PulumiWorkdir) state() string {
	appDir := filepath.Join(w.path, ".pulumi", "stacks", w.project.app.Name)
	path := filepath.Join(appDir, fmt.Sprintf("%v.json", w.project.app.Stage))
	return path
}

func (w *PulumiWorkdir) Export() (*apitype.CheckpointV3, error) {
	var untyped apitype.VersionedCheckpoint
	file, err := os.Open(w.state())
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(file).Decode(&untyped)
	if err != nil {
		return nil, err
	}

	var result apitype.CheckpointV3
	err = json.Unmarshal(untyped.Checkpoint, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (w *PulumiWorkdir) Import(checkpoint *apitype.CheckpointV3) error {
	raw, err := json.MarshalIndent(checkpoint, "", "  ")
	if err != nil {
		return err
	}
	result := apitype.VersionedCheckpoint{
		Version:    3,
		Checkpoint: raw,
	}
	file, err := os.Create(w.state())
	if err != nil {
		return err
	}
	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	err = enc.Encode(result)
	if err != nil {
		return err
	}
	return nil
}

func (w *PulumiWorkdir) EventLogPath() string {
	return filepath.Join(w.path, "eventlog.json")
}
