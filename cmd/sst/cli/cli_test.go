package cli

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgumentList_String(t *testing.T) {
	tests := []struct {
		name     string
		args     ArgumentList
		expected string
	}{
		{
			name:     "empty args",
			args:     ArgumentList{},
			expected: "",
		},
		{
			name: "required args",
			args: ArgumentList{
				{Name: "target", Required: true},
				{Name: "stage", Required: true},
			},
			expected: "<target> <stage>",
		},
		{
			name: "optional args",
			args: ArgumentList{
				{Name: "target", Required: false},
				{Name: "stage", Required: false},
			},
			expected: "[target] [stage]",
		},
		{
			name: "mixed args",
			args: ArgumentList{
				{Name: "target", Required: true},
				{Name: "stage", Required: false},
			},
			expected: "<target> [stage]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.args.String()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGuessStage(t *testing.T) {
	stage := guessStage()
	
	// Should return empty string for protected usernames or a valid stage name
	if stage != "" {
		assert.NotContains(t, []string{"root", "admin", "prod", "dev", "production"}, stage)
		assert.NotEmpty(t, stage)
	}
}

func TestCommand_init(t *testing.T) {
	cmd := &Command{
		Name: "test",
		Flags: []Flag{
			{Name: "string-flag", Type: "string"},
			{Name: "bool-flag", Type: "bool"},
		},
		Children: []*Command{
			{
				Name: "child",
				Flags: []Flag{
					{Name: "child-flag", Type: "string"},
				},
			},
		},
	}

	parsed := make(map[string]interface{})
	cmd.init(parsed)

	// Check that flags were initialized
	assert.Contains(t, parsed, "string-flag")
	assert.Contains(t, parsed, "bool-flag")
	assert.Contains(t, parsed, "child-flag")

	// Check that slices were initialized
	assert.NotNil(t, cmd.Args)
	assert.NotNil(t, cmd.Flags)
	assert.NotNil(t, cmd.Examples)
	assert.NotNil(t, cmd.Children)
}

func TestCli_Positional(t *testing.T) {
	cli := &Cli{
		arguments: []string{"arg1", "arg2", "arg3"},
	}

	assert.Equal(t, "arg1", cli.Positional(0))
	assert.Equal(t, "arg2", cli.Positional(1))
	assert.Equal(t, "arg3", cli.Positional(2))
	assert.Equal(t, "", cli.Positional(10)) // Out of bounds
}

func TestCli_Arguments(t *testing.T) {
	cli := &Cli{
		arguments: []string{"arg1", "arg2"},
	}

	args := cli.Arguments()
	assert.Equal(t, []string{"arg1", "arg2"}, args)
}

func TestCli_Path(t *testing.T) {
	path := CommandPath{
		{Name: "sst"},
		{Name: "deploy"},
	}
	
	cli := &Cli{
		path: path,
	}

	result := cli.Path()
	assert.Len(t, result, 2)
	assert.Equal(t, "sst", result[0].Name)
	assert.Equal(t, "deploy", result[1].Name)
}

func TestCli_Cancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	cli := &Cli{
		Context: ctx,
		cancel:  cancel,
	}

	// Test that Cancel doesn't panic
	assert.NotPanics(t, func() {
		cli.Cancel()
	})

	// Check that context is cancelled
	select {
	case <-cli.Context.Done():
		// Context was cancelled as expected
	default:
		t.Error("Context should be cancelled after calling Cancel()")
	}
}

func TestCli_String(t *testing.T) {
	stringFlag := "test-value"
	cli := &Cli{
		flags: map[string]interface{}{
			"test-flag": &stringFlag,
		},
	}

	assert.Equal(t, "test-value", cli.String("test-flag"))
	assert.Equal(t, "", cli.String("nonexistent"))
}

func TestCli_Bool(t *testing.T) {
	boolFlag := true
	cli := &Cli{
		flags: map[string]interface{}{
			"test-flag": &boolFlag,
		},
	}

	assert.True(t, cli.Bool("test-flag"))
	assert.False(t, cli.Bool("nonexistent"))
}

func TestCli_Env(t *testing.T) {
	env := []string{"VAR1=value1", "VAR2=value2"}
	cli := &Cli{
		env: env,
	}

	result := cli.Env()
	assert.Equal(t, env, result)
}