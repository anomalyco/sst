package project

import (
	"reflect"
	"testing"
)

func TestPulumiCommandArgs(t *testing.T) {
	for _, test := range []struct {
		name    string
		command string
		refresh bool
		want    []string
	}{
		{"remove-default", "remove", false, []string{"destroy", "--yes", "-f"}},
		{"remove-refresh", "remove", true, []string{"destroy", "--yes", "-f", "--refresh"}},
		{"refresh-default", "refresh", false, []string{"refresh", "--yes", "--run-program"}},
		{"refresh-unchanged", "refresh", true, []string{"refresh", "--yes", "--run-program"}},
		{"deploy-default", "deploy", false, []string{"up", "--yes", "-f"}},
		{"deploy-unchanged", "deploy", true, []string{"up", "--yes", "-f"}},
		{"diff-default", "diff", false, []string{"preview"}},
		{"diff-unchanged", "diff", true, []string{"preview"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := pulumiCommandArgs(test.command, test.refresh); !reflect.DeepEqual(got, test.want) {
				t.Fatalf("got %v, want %v", got, test.want)
			}
		})
	}
}
