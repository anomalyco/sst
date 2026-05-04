package provider

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"github.com/sst/sst/v3/internal/util"
)

type snapshotTestHome struct {
	keys []string
	data map[string][]byte
}

func (s *snapshotTestHome) Bootstrap() error {
	return nil
}

func (s *snapshotTestHome) getData(key, app, stage string) (io.Reader, error) {
	data, ok := s.data[key+"/"+app+"/"+stage]
	if !ok {
		return nil, nil
	}
	return bytes.NewReader(data), nil
}

func (s *snapshotTestHome) putData(key, app, stage string, data io.Reader) error {
	return nil
}

func (s *snapshotTestHome) removeData(key, app, stage string) error {
	return nil
}

func (s *snapshotTestHome) setPassphrase(app, stage string, passphrase string) error {
	return nil
}

func (s *snapshotTestHome) getPassphrase(app, stage string) (string, error) {
	return "", nil
}

func (s *snapshotTestHome) listStages(app string) ([]string, error) {
	return nil, nil
}

func (s *snapshotTestHome) listData(key, app, stage string) ([]string, error) {
	return s.keys, nil
}

func (s *snapshotTestHome) cleanup(key, app, stage string) error {
	return nil
}

func (s *snapshotTestHome) info() (util.KeyValuePairs[string], error) {
	return nil, nil
}

func TestLatestValidSnapshotSkipsCorruptedSnapshots(t *testing.T) {
	t.Parallel()

	checkpoint := apitype.CheckpointV3{
		Latest: &apitype.DeploymentV3{Resources: []apitype.ResourceV3{}},
	}
	rawCheckpoint, err := json.Marshal(checkpoint)
	if err != nil {
		t.Fatal(err)
	}
	state, err := json.Marshal(apitype.VersionedCheckpoint{
		Version:    3,
		Checkpoint: rawCheckpoint,
	})
	if err != nil {
		t.Fatal(err)
	}

	home := &snapshotTestHome{
		keys: []string{
			"dev/fe6288d0ec9814453df3c388",
			"dev/fe628cc5182389b648c70130",
		},
		data: map[string][]byte{
			"snapshot/app/dev/fe6288d0ec9814453df3c388": {},
			"snapshot/app/dev/fe628cc5182389b648c70130": state,
		},
	}

	recovered, updateID, err := LatestValidSnapshot(home, "app", "dev")
	if err != nil {
		t.Fatal(err)
	}
	if updateID != "fe628cc5182389b648c70130" {
		t.Fatalf("expected fallback update id, got %q", updateID)
	}
	if !bytes.Equal(recovered, state) {
		t.Fatal("expected recovered snapshot bytes")
	}
}
