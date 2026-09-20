package main

import (
	"context"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/pflag"
	"github.com/sst/sst/v3/cmd/sst/cli"
)

func TestRemoveInput(t *testing.T) {
	for _, test := range []struct {
		name    string
		args    []string
		refresh bool
		targets []string
	}{
		{"default", []string{"remove"}, false, []string{}},
		{"refresh", []string{"remove", "--refresh"}, true, []string{}},
		{"explicit-false", []string{"remove", "--refresh=false"}, false, []string{}},
		{"targeted-refresh", []string{"remove", "--refresh", "--target", "Router,Allowlist"}, true, []string{"Router", "Allowlist"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			originalArgs, originalFlags := os.Args, pflag.CommandLine
			defer func() { os.Args, pflag.CommandLine = originalArgs, originalFlags }()
			os.Args = append([]string{"sst"}, test.args...)
			pflag.CommandLine = pflag.NewFlagSet("sst", pflag.ContinueOnError)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			c, err := cli.New(ctx, cancel, root, "dev")
			if err != nil {
				t.Fatal(err)
			}
			input := removeInput(c, 1234)
			if input.Command != "remove" || input.Refresh != test.refresh || input.ServerPort != 1234 || !reflect.DeepEqual(input.Target, test.targets) {
				t.Fatalf("unexpected removal input: %#v", input)
			}
		})
	}
}
