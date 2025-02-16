package state

import (
	"context"
	"encoding/json"
	"os"

	"github.com/pulumi/pulumi/pkg/v3/resource/stack"
	"github.com/pulumi/pulumi/pkg/v3/secrets"
	"github.com/pulumi/pulumi/pkg/v3/secrets/passphrase"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
)

func Decrypt(ctx context.Context, passphrase string, checkpoint *apitype.CheckpointV3) (*apitype.CheckpointV3, error) {
	os.Setenv("PULUMI_CONFIG_PASSPHRASE", passphrase)
	sp := &defaultSecretsProvider{
		passphrase: passphrase,
	}
	snapshot, err := stack.DeserializeCheckpoint(ctx, sp, checkpoint)
	if err != nil {
		return nil, err
	}
	depl, err := stack.SerializeDeployment(ctx, snapshot, true)
	if err != nil {
		return nil, err
	}
	checkpoint.Latest = depl
	return &apitype.CheckpointV3{
		Stack:  checkpoint.Stack,
		Latest: depl,
	}, nil
}

type defaultSecretsProvider struct {
	passphrase string
}

func (d *defaultSecretsProvider) OfType(ty string, state json.RawMessage) (secrets.Manager, error) {
	sm, err := passphrase.NewPromptingPassphraseSecretsManagerFromState(state)
	if err != nil {
		return nil, err
	}
	return sm, nil
}
